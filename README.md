# k8s-local-bench

## Setup a k8s cluster to work locally

```bash
mkdir my-local-workspace
k8s-local-bench cluster create
```

# TODO

- [ ] rename to project to localplane
- [ ] update the documentation on all the new features / concepts + add a clead README with quickstart
- [ ] automatically get the ingress ip (only load balancer service) + update dnsmasq.conf + restart
- [ ] add mermaid diagrams explaining how it works
- [ ] add some spinners and logs to the create command
- [ ] replace prompts with promptui
- [ ] use https://github.com/jedib0t/go-pretty for tables, and progress display instead of basic logging
- [ ] add delete validation to the delete command
- [ ] display argocd and headlamp (link + token) when the cluster is created
- [ ] release new version of the k8s-local-bench chart
- [ ] configure victoria-metrics ingress