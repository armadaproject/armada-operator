package common

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestCheckAndHandleResourceDeletion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create a logger (for tests, we use logr.Discard() to avoid actual logging)
	logger := logr.Discard()

	// Table-driven test cases
	tests := []struct {
		name              string
		deletionTimestamp *time.Time
		finalizerPresent  bool
		expectFinalizer   bool
		expectFinish      bool
		expectError       bool
	}{
		{
			name:              "Object not being deleted, finalizer not present",
			deletionTimestamp: nil,
			finalizerPresent:  false,
			expectFinalizer:   true,
			expectFinish:      false,
			expectError:       false,
		},
		{
			name:              "Object not being deleted, finalizer already present",
			deletionTimestamp: nil,
			finalizerPresent:  true,
			expectFinalizer:   true,
			expectFinish:      false,
			expectError:       false,
		},
		{
			name:              "Object being deleted, finalizer present",
			deletionTimestamp: ptr.To(time.Now()),
			finalizerPresent:  true,
			expectFinalizer:   false,
			expectFinish:      true,
			expectError:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake object
			object := newTestObject(t)

			// Set DeletionTimestamp if the test requires it
			if tc.deletionTimestamp != nil {
				object.SetDeletionTimestamp(&metav1.Time{Time: *tc.deletionTimestamp})
			}

			// Add or remove finalizer based on the test case
			finalizer := "test.finalizer"
			if tc.finalizerPresent {
				controllerutil.AddFinalizer(object, finalizer)
			}

			// Create a fake client
			fakeClient := fake.NewClientBuilder().WithObjects(object).Build()

			cleanupFunc := func(ctx context.Context) error {
				if tc.deletionTimestamp == nil {
					t.Fatalf("cleanup function should not be called")
				}
				return nil
			}

			// Call the function under test
			finish, err := CheckAndHandleObjectDeletion(ctx, fakeClient, object, finalizer, cleanupFunc, logger)

			// Check for errors
			if (err != nil) != tc.expectError {
				t.Errorf("Expected error: %v, got: %v", tc.expectError, err)
			}

			// Check if reconciliation should finish
			if finish != tc.expectFinish {
				t.Errorf("Expected finish: %v, got: %v", tc.expectFinish, finish)
			}

			// Check if finalizer was added/removed as expected
			hasFinalizer := controllerutil.ContainsFinalizer(object, finalizer)
			if hasFinalizer != tc.expectFinalizer {
				t.Errorf("Expected finalizer: %v, got: %v", tc.expectFinalizer, hasFinalizer)
			}
		})
	}
}

func TestGetObjectFromCache(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// No-op logger for testing
	logger := logr.Discard()

	// Test cases
	tests := []struct {
		name         string
		objectExists bool
		expectMiss   bool
		expectError  bool
		returnError  error
	}{
		{
			name:         "Object exists in cache",
			objectExists: true,
			returnError:  nil,
			expectMiss:   false,
			expectError:  false,
		},
		{
			name:         "Object not found in cache",
			objectExists: false,
			returnError:  k8serrors.NewNotFound(newTestGroupResource(t), "test-resource"),
			expectMiss:   true,
			expectError:  false,
		},
		{
			name:         "Error while fetching from cache",
			objectExists: false,
			returnError:  errors.New("some network error"),
			expectMiss:   true,
			expectError:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake client with the expected error behavior
			clientBuilder := fake.NewClientBuilder()

			// Create a fake object
			object := newTestObject(t)

			if tc.objectExists {
				clientBuilder.WithObjects(object)
			}

			if tc.expectError {
				clientBuilder.WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						return tc.returnError
					},
				})
			}

			fakeClient := clientBuilder.Build()

			// Call the function under test
			namespacedName := types.NamespacedName{Name: "test-resource", Namespace: "default"}
			miss, err := GetObject(ctx, fakeClient, object, namespacedName, logger)
			if tc.expectError {
				assert.ErrorIs(t, err, tc.returnError)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, miss, tc.expectMiss)
		})
	}
}

// newTestObject returns a test unstructured.Unstructured object only to be used in tests.
func newTestObject(t *testing.T) *unstructured.Unstructured {
	object := &unstructured.Unstructured{}
	object.SetNamespace("default")
	object.SetName("test-resource")
	object.SetUID(uuid.NewUUID())
	object.SetGroupVersionKind(newTestGroupVersionKind(t))
	return object
}

// newTestGroupVersionKind returns a test schema.GroupVersionKind only to be used in tests.
func newTestGroupVersionKind(t *testing.T) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "test.group",
		Version: "v1",
		Kind:    "TestKind",
	}
}

// newTestGroupResource returns a test schema.GroupResource only to be used in tests.
func newTestGroupResource(t *testing.T) schema.GroupResource {
	return schema.GroupResource{
		Group:    "test.group",
		Resource: "test-resource",
	}
}
