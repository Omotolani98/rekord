package redact

import "regexp"

type Pattern struct {
	Category    string
	Re          *regexp.Regexp
	Replacement string
}

func DefaultPatterns() []Pattern {
	const redacted = "[REDACTED]"
	keyValue := func(category, key string) Pattern {
		return Pattern{
			Category:    category,
			Re:          regexp.MustCompile(`(?i)\b(` + key + `)=\S+`),
			Replacement: "${1}=" + redacted,
		}
	}
	token := func(category, expr string) Pattern {
		return Pattern{
			Category:    category,
			Re:          regexp.MustCompile(expr),
			Replacement: redacted,
		}
	}

	return []Pattern{
		keyValue("env-secret", "OPENAI_API_KEY|GITHUB_TOKEN|AWS_SECRET_ACCESS_KEY|AWS_ACCESS_KEY_ID|DATABASE_URL"),
		token("openai-key", `sk-[A-Za-z0-9]{16,}`),
		token("github-token", `ghp_[A-Za-z0-9]{20,}`),
		token("aws-access-key", `AKIA[0-9A-Z]{16}`),
		token("postgres-url", `postgres://\S+`),
		token("mysql-url", `mysql://\S+`),
		keyValue("password", "password"),
		keyValue("token", "token"),
		keyValue("secret", "secret"),
	}
}
