# Go Backend Template

A reusable Go backend template with:

- Go
- Docker
- Docker Compose
- PostgreSQL
- Layered architecture (Handler → Service → Repository)
- SQL migrations

---

## Creating a New Project

1. Copy this repository:  cp -R GoStarter <NEW_PROJECT>

2. make a new git repo: 
	cd <NEW_DIR>
	rm -rf .git
	git init
	git add .
	git commit -m "Initial commit"
	git remote add origin https://github.com/<your-username>/<NEW_PROJECT>.git
	git branch -M main
	git push -u origin main

3. Update the Go module.

```bash
go mod edit -module github.com/<username>/<project-name>
go mod tidy
```

4. Update project configuration.

- Update `.env`
- Update database name (`POSTGRES_DB`)
- Update database user/password if desired
- Rename the Docker image (optional)
- Rename any service names in `docker-compose.yml` (optional)

5. Create your first database migration.

Example:

migrations/
└── 001_init.sql

6. Replace the example code.

- Add models
- Add repositories
- Add services
- Add handlers
- Register routes
- Remove the sample "Hello World" endpoint

7. Start the project.

```bash
docker compose up --build
```

8. Verify everything works.

- Visit http://localhost:8080
- Check the health endpoint
- Confirm PostgreSQL connected successfully
