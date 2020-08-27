package util

type MeConfig struct {
	MQconf RabbitMQ `yaml:"rabbitmq"`
	CacheConf Cache `yaml:"redis"`
	GRPCConf GRPC `yaml:"grpc"`
	MeConf Mengine `yaml:"gomengine"`
}

type Cache struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	Password string `yaml:"password"`
}

type RabbitMQ struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type GRPC struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type Mengine struct {
	Accuracy int `yaml:"accuracy"`
}