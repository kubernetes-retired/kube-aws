package model

type SystemdMessageResponse struct {
	InstanceId  string `json:"instanceId"`
	Hostname    string `json:"hostname"`
	CmdName     string `json:"cmdName"`
	Exe         string `json:"exe"`
	CmdLine     string `json:"cmdLine"`
	SystemdUnit string `json:"systemdUnit"`
	Priority    string `json:"priority"`
	Message     string `json:"message"`
}
