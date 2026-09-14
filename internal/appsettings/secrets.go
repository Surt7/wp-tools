package appsettings

import "fmt"

type SecretDefinition struct {
	Key   string
	Label string
}

func DefaultSecrets() []SecretDefinition {
	return []SecretDefinition{
		{
			Key:   "Keycloak:ClientSecret",
			Label: "Keycloak client secret",
		},
		{
			Key:   "Redis:Password",
			Label: "Redis password",
		},
		{
			Key:   "PostGres:Password",
			Label: "PostGres password",
		},
		{
			Key:   "RabbitMQ:Password",
			Label: "RabbitMQ password",
		},
		{
			Key:   "Kafka:Password",
			Label: "Kafka password",
		},
	}
}

func DisplaySecrets(secrets []SecretDefinition) {
	for _, secret := range secrets {
		fmt.Println("-------")
		fmt.Println(secret.Label + " >>> ")
	}
}
