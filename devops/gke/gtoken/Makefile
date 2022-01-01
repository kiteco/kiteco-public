PROJECT=kite-dev-XXXXXXX
GCR_REPO_URL=gcr.io/$(PROJECT)
CONTEXT := gke_$(PROJECT)_$(REGION)_$(CLUSTER)
CA_BUNDLE := $(shell kubectl config view --raw --flatten -o json | jq -r '.clusters[] | select(.name == "$(CONTEXT)") | .cluster."certificate-authority-data"')
GTOKEN_WEBHOOK_TAG=v0.3.5

all: kubectl deployment.create_cert deployment.apply

gtoken-webhook:
	docker pull doitintl/gtoken-webhook:$(GTOKEN_WEBHOOK_TAG)
	docker tag doitintl/gtoken-webhook:$(GTOKEN_WEBHOOK_TAG) $(GCR_REPO_URL)/gtoken-webhook:$(GTOKEN_WEBHOOK_TAG)
	docker push $(GCR_REPO_URL)/gtoken-webhook:$(GTOKEN_WEBHOOK_TAG)

build/resources.yaml: resources.yaml
	ytt --output-files build -f resources.yaml -f data.yaml -v ca_bundle=$(CA_BUNDLE) -v gtoken_webhook_image=$(GCR_REPO_URL)/gtoken-webhook:$(GTOKEN_WEBHOOK_TAG)

deployment.create_cert:
	./webhook-create-signed-cert.sh

deployment.apply: build/resources.yaml gtoken-webhook
	kubectl apply -f build/resources.yaml

kubectl:
	gcloud container clusters get-credentials $(CLUSTER) --region $(REGION) --project $(PROJECT)
	kubectl config use-context $(CONTEXT)
