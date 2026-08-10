.PHONY: init fmt validate plan apply destroy kubeconfig kubeconfig-env alias-k nodes

KUBECONFIG_PATH := ~/.kube/paas-ske-config

init:
	terraform init

fmt:
	terraform fmt -recursive

validate:
	terraform validate

plan:
	terraform plan

apply:
	terraform apply

destroy:
	terraform destroy

## Writes the kubeconfig to disk. Does NOT export it into your shell —
## see the note above on why a Makefile recipe can't do that directly.
kubeconfig:
	@terraform output -raw kubeconfig > $(KUBECONFIG_PATH)
	@echo "Kubeconfig written to $(KUBECONFIG_PATH)"
	@echo "Run manually: export KUBECONFIG=$(KUBECONFIG_PATH)"

## Writes the file AND prints an export line you can eval directly
## into your current shell — this is how you actually get the export
## to "stick", by having your own shell run it, not make's subshell.
kubeconfig-env:
	@terraform output -raw kubeconfig > $(KUBECONFIG_PATH)
	@echo "export KUBECONFIG=$(KUBECONFIG_PATH)"

## Appends the k=kubectl alias to your zsh config (idempotent — won't
## duplicate if you run it twice). Still requires a `source` after.
alias-k:
	@grep -qxF "alias k=kubectl" $(HOME)/.zshrc || echo "alias k=kubectl" >> $(HOME)/.zshrc
	@echo "Added to ~/.zshrc — run: source ~/.zshrc"

nodes:
	kubectl get nodes -o wide