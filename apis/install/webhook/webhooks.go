package webhook

import ctrl "sigs.k8s.io/controller-runtime"

// Setup sets up the webhooks for core controllers. It returns the name of the
// webhook that failed to create and an error, if any.
func Setup(mgr ctrl.Manager) (string, error) {
	if err := SetupWebhookForBinoculars(mgr); err != nil {
		return "Binoculars", err
	}

	if err := SetupWebhookForEventIngester(mgr); err != nil {
		return "EventIngester", err
	}

	if err := SetupWebhookForArmadaServer(mgr); err != nil {
		return "ArmadaServer", err
	}

	if err := SetupWebhookForExecutor(mgr); err != nil {
		return "Executor", err
	}

	if err := SetupWebhookForLookout(mgr); err != nil {
		return "Lookout", err
	}
	return "", nil
}
