package generator

import "fmt"

func GitHubActions(appName, serverHost, serverUser, serverKeySecret, appDir string) string {
	if appDir == "" {
		appDir = fmt.Sprintf("/opt/ship/%s", appName)
	}
	return fmt.Sprintf(`name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    name: Deploy to VPS
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Deploy via SSH
        uses: appleboy/ssh-action@v1.0.3
        with:
          host: ${{ secrets.%s }}
          username: %s
          key: ${{ secrets.%s }}
          script: |
            cd %s
            git pull origin main
            docker compose pull
            docker compose up -d --build
            docker compose exec -T app sh -c "your-migration-command || true"
            docker image prune -f
`, serverKeySecret+"_HOST", serverUser, serverKeySecret, appDir)
}
