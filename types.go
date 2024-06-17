package main

type Config struct {
	Identities    map[string]Identity     `json:"identities"`
	HostDefaults  HostDefaults            `json:"host_defaults"`
	HostTemplates map[string]HostTemplate `json:"host_templates"`
	Checks        map[string]Check        `json:"checks"`
	HostGroups    map[string]HostGroup    `json:"host_groups"`
	Report        Report                  `json:"report"`
}

type Identity struct {
	User       string `json:"user"`
	Key        string `json:"key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
	Password   string `json:"password,omitempty"`
}

type HostDefaults struct {
	Identity   string            `json:"identity"`
	HostVars   map[string]string `json:"host_vars"`
	HostChecks []string          `json:"host_checks"`
}

type HostTemplate struct {
	HostVars   map[string]string `json:"host_vars,omitempty"`
	HostChecks []string          `json:"host_checks"`
}

type HostGroup struct {
	HostVars map[string]string `json:"host_vars,omitempty"`
	Hosts    map[string]Host   `json:"hosts"`
}

type Check struct {
	Title       string      `json:"title"`
	Command     string      `json:"command,omitempty"`
	Service     string      `json:"service,omitempty"`
	URL         string      `json:"url,omitempty"`
	FailWhen    string      `json:"fail_when"`
	FailValue   interface{} `json:"fail_value"` // Can be a string or a list of strings
	Description string      `json:"description,omitempty"`
	Graph       GraphConfig `json:"graph,omitempty"`
	Local       bool        `json:"local,omitempty"`
}

type Host struct {
	Identity     string            `json:"identity,omitempty"`
	HostTemplate string            `json:"host_template,omitempty"`
	HostVars     map[string]string `json:"host_vars,omitempty"`
	HostChecks   []string          `json:"host_checks"`
}

type Report struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	CSS         string `json:"css"`
}

type GraphConfig struct {
	Title  string              `json:"title"`
	Type   string              `json:"type"`
	Show   bool                `json:"show"`
	Legend bool                `json:"legend"`
	Colors map[string][]string `json:"colors"`
}

type CheckResult struct {
	Host      string            `json:"host"`
	Check     string            `json:"check"`
	Status    string            `json:"status"`
	Value     string            `json:"value"`
	Timestamp string            `json:"timestamp"`
	Vars      map[string]string `json:"vars,omitempty"`
}

type ResultFile struct {
	Checks  map[string]Check                    `json:"checks"`
	Results map[string]map[string]CheckResult `json:"results"`
}
