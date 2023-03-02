package planner

type IscsiContainerArgs struct {
	planner *Planner
}

func (pl *Planner) Args() *IscsiContainerArgs {
	return &IscsiContainerArgs{pl}
}

func (i *IscsiContainerArgs) Initializer(cmd string) []string {
	args := []string{}
	if i.planner.IsClustered() {
		args = append(args, "--skip-if-already-init")
	}
	args = append(args, cmd)
	return args
}

func (i *IscsiContainerArgs) SetNode() []string {
	args := []string{
		"set-node",
		"--hostname=$(HOSTNAME)",
	}

	return args
}

func (*IscsiContainerArgs) UpdateConfigWatch() []string {
	return []string{
		"update-config",
		"--watch",
	}
}

func (i *IscsiContainerArgs) Run(name string) []string {
	args := []string{}
	return args
}
