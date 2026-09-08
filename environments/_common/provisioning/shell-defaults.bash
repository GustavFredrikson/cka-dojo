# dojo shell defaults
alias k=kubectl
type -t __start_kubectl >/dev/null || source <(kubectl completion bash)
complete -o default -F __start_kubectl k
export do='--dry-run=client -o yaml'
export now='--force --grace-period=0'
