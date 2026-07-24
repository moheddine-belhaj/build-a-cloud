.PHONY: init validate plan apply destroy fmt ssh output clean

init:
	terraform init

fmt:
	terraform fmt -recursive

validate: validate
	terraform validate

plan: plan
	terraform plan

apply: apply
	terraform apply -auto-approve

destroy:
	terraform destroy -auto-approve

output:
	terraform output

ssh:
	ssh -i ~/.ssh/tf_key ubuntu@$$(terraform output -raw floating_ips | tr -d '[]"' | head -1)

clean:
	rm -rf .terraform .terraform.lock.hcl terraform.tfstate terraform.tfstate.backup
