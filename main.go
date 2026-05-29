package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

type GitHubContent struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

func parseEnv(content string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}
	return result
}

func authMiddleware(c *fiber.Ctx) error {
	apiKey := c.Get("X-API-Key")
	if apiKey != os.Getenv("API_KEY") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}
	return c.Next()
}

func getConfig(c *fiber.Ctx) error {
	projectName := c.Params("projectName")
	filePath := fmt.Sprintf("%s.env", projectName)

	githubUsername := os.Getenv("GITHUB_USERNAME")
	repoName := os.Getenv("REPO_NAME")
	branchName := os.Getenv("BRANCH_NAME")
	githubToken := os.Getenv("GITHUB_TOKEN")

	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		githubUsername, repoName, filePath, branchName,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create request",
		})
	}

	req.Header.Set("Authorization", "token "+githubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	log.Println("URL:", url)
	resp, err := client.Do(req)
	if err != nil {
		log.Println("HTTP ERROR:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch config",
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("❌ GitHub API error: %s", string(body))
		return c.Status(resp.StatusCode).JSON(fiber.Map{
			"error": "Config not found",
		})
	}

	var githubContent GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&githubContent); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode response",
		})
	}

	decoded, err := base64.StdEncoding.DecodeString(
		strings.ReplaceAll(githubContent.Content, "\n", ""),
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode content",
		})
	}

	parsed := parseEnv(string(decoded))
	return c.Status(fiber.StatusOK).JSON(parsed)
}

func main() {
	log.Println("API_KEY:", os.Getenv("API_KEY"))
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	app := fiber.New(fiber.Config{
		AppName: "SCC Service",
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
		})
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome to SCC Service")
	})

	app.Get("/config/:projectName", authMiddleware, getConfig)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Printf("SCC Service running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
