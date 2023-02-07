package conf

var globalConf *OperatorConfig

func Get() *OperatorConfig {
	return globalConf
}

func Load(s *Source) error {
	c, err := s.Read()
	if err != nil {
		return err
	}
	globalConf = c
	return nil
}
