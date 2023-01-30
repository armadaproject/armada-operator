kind create cluster
make dev-setup
DISABLE_WEBHOOKS=true make install
sleep 10
kubectl apply -f ./config/samples/install_v1alpha1_lookoutingester.yaml
