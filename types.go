package main

type VarMap map[string]interface{}

type Config struct {
	Identities    map[string]Identity     `json:"identities" yaml:"identities"`
	HostDefaults  HostDefaults            `json:"host_defaults" yaml:"host_defaults"`
	HostTemplates map[string]HostTemplate `json:"host_templates" yaml:"host_templates"`
	CheckGroups   map[string]CheckGroup   `json:"check_groups" yaml:"check_groups"`
	Checks        map[string]Check        `json:"checks" yaml:"checks"`
	URLChecks     map[string]Check        `json:"url_checks,omitempty" yaml:"url_checks,omitempty"`
	HostGroups    map[string]HostGroup    `json:"host_groups" yaml:"host_groups"`
	Report        Report                  `json:"report" yaml:"report"`
	Auth          AuthConfig              `json:"auth,omitempty" yaml:"auth,omitempty"`
}

type AuthConfig struct {
	Mode           string   `json:"mode,omitempty" yaml:"mode,omitempty"`
	UserHeader     string   `json:"user_header,omitempty" yaml:"user_header,omitempty"`
	EmailHeader    string   `json:"email_header,omitempty" yaml:"email_header,omitempty"`
	GroupsHeader   string   `json:"groups_header,omitempty" yaml:"groups_header,omitempty"`
	LogoutPath     string   `json:"logout_path,omitempty" yaml:"logout_path,omitempty"`
	ViewerGroups   []string `json:"viewer_groups,omitempty" yaml:"viewer_groups,omitempty"`
	OperatorGroups []string `json:"operator_groups,omitempty" yaml:"operator_groups,omitempty"`
	AdminGroups    []string `json:"admin_groups,omitempty" yaml:"admin_groups,omitempty"`
}

type Identity struct {
	User       string `json:"user" yaml:"user"`
	Key        string `json:"key,omitempty" yaml:"key,omitempty"`
	Passphrase string `json:"passphrase,omitempty" yaml:"passphrase,omitempty"`
	Password   string `json:"password,omitempty" yaml:"password,omitempty"`
}

type HostDefaults struct {
	Identity    string   `json:"identity" yaml:"identity"`
	HostVars    VarMap   `json:"host_vars" yaml:"host_vars"`
	HostChecks  []string `json:"host_checks" yaml:"host_checks"`
	CheckGroups []string `json:"check_groups,omitempty" yaml:"check_groups,omitempty"`
}

type HostTemplate struct {
	HostVars    VarMap   `json:"host_vars,omitempty" yaml:"host_vars,omitempty"`
	HostChecks  []string `json:"host_checks" yaml:"host_checks"`
	CheckGroups []string `json:"check_groups,omitempty" yaml:"check_groups,omitempty"`
}

type HostGroup struct {
	HostVars    VarMap          `json:"host_vars,omitempty" yaml:"host_vars,omitempty"`
	HostChecks  []string        `json:"host_checks,omitempty" yaml:"host_checks,omitempty"`
	CheckGroups []string        `json:"check_groups,omitempty" yaml:"check_groups,omitempty"`
	Hosts       map[string]Host `json:"hosts" yaml:"hosts"`
}

type CheckGroup struct {
	Vars   VarMap   `json:"vars,omitempty" yaml:"vars,omitempty"`
	Checks []string `json:"checks" yaml:"checks"`
}

type Check struct {
	Title            string          `json:"title" yaml:"title"`
	Command          string          `json:"command,omitempty" yaml:"command,omitempty"`
	Service          string          `json:"service,omitempty" yaml:"service,omitempty"`
	URL              string          `json:"url,omitempty" yaml:"url,omitempty"`
	FollowRedirects  bool            `json:"follow_redirects,omitempty" yaml:"follow_redirects,omitempty"`
	ExpectedContains string          `json:"expected_contains,omitempty" yaml:"expected_contains,omitempty"`
	FailWhen         string          `json:"fail_when" yaml:"fail_when"`
	FailValue        interface{}     `json:"fail_value" yaml:"fail_value"` // Can be a string or a list of strings
	Description      string          `json:"description,omitempty" yaml:"description,omitempty"`
	Graph            GraphConfig     `json:"graph,omitempty" yaml:"graph,omitempty"`
	Sparkline        SparklineConfig `json:"sparkline,omitempty" yaml:"sparkline,omitempty"`
	Timeout          string          `json:"timeout,omitempty" yaml:"timeout,omitempty"` // Store as string
	Local            bool            `json:"local,omitempty" yaml:"local,omitempty"`
	Vars             VarMap          `json:"vars,omitempty" yaml:"vars,omitempty"`
}

type Host struct {
	Identity     string   `json:"identity,omitempty" yaml:"identity,omitempty"`
	HostTemplate string   `json:"host_template,omitempty" yaml:"host_template,omitempty"`
	HostVars     VarMap   `json:"host_vars,omitempty" yaml:"host_vars,omitempty"`
	HostChecks   []string `json:"host_checks" yaml:"host_checks"`
	CheckGroups  []string `json:"check_groups,omitempty" yaml:"check_groups,omitempty"`
}

type Report struct {
	Title       string `json:"title" yaml:"title"`
	Subtitle    string `json:"subtitle" yaml:"subtitle"`
	Description string `json:"description" yaml:"description"`
	Copyright   string `json:"copyright" yaml:"copyright"`
	CSS         string `json:"css" yaml:"css"`
}

type GraphConfig struct {
	Title  string              `json:"title" yaml:"title"`
	Type   string              `json:"type" yaml:"type"`
	Show   bool                `json:"show" yaml:"show"`
	Legend bool                `json:"legend" yaml:"legend"`
	Colors map[string][]string `json:"colors" yaml:"colors"`
}

type SparklineConfig struct {
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Points  int  `json:"points,omitempty" yaml:"points,omitempty"`
}

type CheckResult struct {
	Host         string            `json:"host"`
	Check        string            `json:"check"`
	Status       string            `json:"status"`
	Value        string            `json:"value"`
	Timestamp    string            `json:"timestamp"`
	Vars         map[string]string `json:"vars,omitempty"`
	URL          string            `json:"url,omitempty"`
	StatusCode   int               `json:"status_code,omitempty"`
	LatencyMs    int64             `json:"latency_ms,omitempty"`
	Redirected   bool              `json:"redirected,omitempty"`
	Location     string            `json:"location,omitempty"`
	FinalURL     string            `json:"final_url,omitempty"`
	ErrorType    string            `json:"error_type,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
}

type ResultFile struct {
	Checks      map[string]Check                  `json:"checks"`
	Results     map[string]map[string]CheckResult `json:"results"`
	URLChecks   map[string]Check                  `json:"url_checks,omitempty"`
	URLResults  map[string]CheckResult            `json:"url_results,omitempty"`
	Report      Report                            `json:"report"` // Voeg de report-informatie toe
	Status      string                            `json:"status,omitempty"`
	Errors      []string                          `json:"errors,omitempty"`
	GeneratedAt string                            `json:"generated_at,omitempty"`
}

type TargetedRunRequest struct {
	Kind      string `json:"kind"`
	CheckName string `json:"check_name"`
	Host      string `json:"host,omitempty"`
}

type HistoryRunMetadata struct {
	RunType     string
	TargetKind  string
	TargetName  string
	TargetScope string
}

type HistoryRun struct {
	ID           int64          `json:"id"`
	GeneratedAt  string         `json:"generated_at"`
	Status       string         `json:"status"`
	RunType      string         `json:"run_type,omitempty"`
	TargetKind   string         `json:"target_kind,omitempty"`
	TargetName   string         `json:"target_name,omitempty"`
	TargetScope  string         `json:"target_scope,omitempty"`
	HostCount    int            `json:"host_count"`
	CheckCount   int            `json:"check_count"`
	PassedCount  int            `json:"passed_count"`
	FailedCount  int            `json:"failed_count"`
	DurationMs   int64          `json:"duration_ms"`
	ErrorSummary map[string]int `json:"error_summary"`
}

type TargetedRunResponse struct {
	Run         HistoryRun    `json:"run"`
	HostResults []CheckResult `json:"host_results,omitempty"`
	URLResults  []CheckResult `json:"url_results,omitempty"`
}

type HistoryEventRecord struct {
	ID           int64  `json:"id"`
	RunID        int64  `json:"run_id"`
	EventTime    string `json:"event_time"`
	EventType    string `json:"event_type"`
	Host         string `json:"host"`
	CheckName    string `json:"check_name"`
	Status       string `json:"status"`
	Value        string `json:"value"`
	ErrorType    string `json:"error_type,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type HistorySparklineMetric struct {
	RunID       int64   `json:"run_id"`
	GeneratedAt string  `json:"generated_at"`
	Host        string  `json:"host"`
	CheckName   string  `json:"check_name"`
	Value       float64 `json:"value"`
	Status      string  `json:"status"`
}

type CheckHistoryDetail struct {
	Host      string                   `json:"host"`
	CheckName string                   `json:"check_name"`
	Metrics   []HistorySparklineMetric `json:"metrics"`
	Events    []HistoryEventRecord     `json:"events"`
}

type PreflightCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type PreflightReport struct {
	OverallStatus string           `json:"overall_status"`
	ConfigPath    string           `json:"config_path"`
	WorkingDir    string           `json:"working_dir"`
	Checks        []PreflightCheck `json:"checks"`
}

type AuthPermissions struct {
	View    bool `json:"view"`
	Operate bool `json:"operate"`
	Admin   bool `json:"admin"`
}

type AuthSession struct {
	Mode          string          `json:"mode"`
	Authenticated bool            `json:"authenticated"`
	Username      string          `json:"username,omitempty"`
	Email         string          `json:"email,omitempty"`
	Groups        []string        `json:"groups,omitempty"`
	Role          string          `json:"role"`
	LogoutURL     string          `json:"logout_url,omitempty"`
	Permissions   AuthPermissions `json:"permissions"`
}
