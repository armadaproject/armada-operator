package builders

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateClusterRole(crdName string, crdNamespace string) *rbacv1.ClusterRole {
	podRules := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch", "create", "delete", "deletecollection", "patch", "update"},
		APIGroups: []string{""},
		Resources: []string{"pods"},
	}
	eventRules := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch", "delete", "deletecollection", "patch"},
		APIGroups: []string{""},
		Resources: []string{"events"},
	}
	serviceRules := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch", "create", "delete", "deletecollection"},
		APIGroups: []string{""},
		Resources: []string{"services"},
	}
	nodeRules := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch"},
		APIGroups: []string{""},
		Resources: []string{"nodes"},
	}
	nodeProxyRules := rbacv1.PolicyRule{
		Verbs:     []string{"get"},
		APIGroups: []string{""},
		Resources: []string{"node/proxy"},
	}
	userRules := rbacv1.PolicyRule{
		Verbs:     []string{"impersonate"},
		APIGroups: []string{""},
		Resources: []string{"users", "groups"},
	}
	ingressRules := rbacv1.PolicyRule{
		Verbs:     []string{"get", "list", "watch", "create", "delete", "deletecollection"},
		APIGroups: []string{"networking.k8s.io"},
		Resources: []string{"ingresses"},
	}
	tokenRules := rbacv1.PolicyRule{
		Verbs:     []string{"create"},
		APIGroups: []string{""},
		Resources: []string{"serviceaccounts/token"},
	}
	tokenReviewRules := rbacv1.PolicyRule{
		Verbs:     []string{"create"},
		APIGroups: []string{"authentication.k8s.io"},
		Resources: []string{"tokenreviews"},
	}
	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: crdName, Namespace: crdNamespace},
		Rules:      []rbacv1.PolicyRule{podRules, eventRules, serviceRules, nodeRules, nodeProxyRules, userRules, ingressRules, tokenRules, tokenReviewRules},
	}
	return &clusterRole
}
