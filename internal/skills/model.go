package skills

type Step struct {
	Run string `yaml:"run"`
}

type Skill struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Steps       []Step `yaml:"steps"`
	SourcePath  string `yaml:"-"`
}
