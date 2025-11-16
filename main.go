package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Question struct {
	Correct int      `json:"correct"`
	Text    string   `json:"text"`
	Options []string `json:"options"`
}

type Quiz struct {
	Questions []Question
	StartTime time.Time
}

type Session struct {
	Quiz     *Quiz
	Username string
}

var quizzes = make(map[string]*Session)

func loadQuestions(filename string) ([]Question, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.Comma = ';'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var questions []Question
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		correct, _ := strconv.Atoi(strings.TrimSpace(line[0]))
		text := strings.TrimSpace(line[1])
		options := make([]string, 0, len(line)-2)
		for _, opt := range line[2:] {
			options = append(options, strings.TrimSpace(opt))
		}
		if correct > 0 && correct <= len(options) {
			questions = append(questions, Question{Correct: correct, Text: text, Options: options})
		}
	}
	return questions, nil
}

func writeLog(username string, correct, total int, duration time.Duration) {
	f, err := os.OpenFile("quiz.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("log error:", err)
		return
	}
	defer f.Close()

	now := time.Now().Format("2006-01-02|15:04:05")
	minutes := int(duration.Minutes())
	seconds := int(duration.Seconds()) % 60
	timeStr := fmt.Sprintf("%02d:%02d", minutes, seconds)

	line := fmt.Sprintf("%s, %s, правильных ответов %d из %d, время: %s\n", now, username, correct, total, timeStr)
	if _, err := f.WriteString(line); err != nil {
		log.Println("write log error:", err)
	}
}

func main() {
	r := gin.Default()
	t := template.New("").Funcs(template.FuncMap{
		"json": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
	})
	r.SetHTMLTemplate(template.Must(t.ParseFiles(
		"templates/start.html",
		"templates/quiz.html",
		"templates/result.html",
	)))

	r.Static("/static", "./static")

	questions, err := loadQuestions("questions.csv")
	if err != nil {
		panic(err)
	}

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "start.html", gin.H{
			"Title": "Тестирование",
		})
	})

	r.POST("/start", func(c *gin.Context) {
		name := c.PostForm("name")
		if name == "" {
			name = "Аноним"
		}

		sessionID := fmt.Sprintf("%d", time.Now().UnixNano())
		quizzes[sessionID] = &Session{
			Quiz: &Quiz{
				Questions: questions,
				StartTime: time.Now(),
			},
			Username: name,
		}

		c.SetCookie("session", sessionID, 3600, "/", "", false, true)
		c.Redirect(http.StatusSeeOther, "/quiz")
	})

	r.GET("/quiz", func(c *gin.Context) {
		session, _ := c.Cookie("session")
		s := quizzes[session]
		if s == nil {
			c.Redirect(http.StatusSeeOther, "/")
			return
		}

		c.HTML(http.StatusOK, "quiz.html", gin.H{
			"Questions": s.Quiz.Questions,
			"Total":     len(s.Quiz.Questions),
		})
	})

	r.POST("/submit", func(c *gin.Context) {
		session, _ := c.Cookie("session")
		s := quizzes[session]
		if s == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Сессия истекла"})
			return
		}
		quiz := s.Quiz
		username := s.Username

		answers := make([]int, len(quiz.Questions))
		for i := range quiz.Questions {
			key := fmt.Sprintf("q%d", i)
			val := c.PostForm(key)
			if val != "" {
				ans, _ := strconv.Atoi(val)
				answers[i] = ans
			}
		}

		correctCount := 0
		duration := time.Since(quiz.StartTime)

		var resultBuilder strings.Builder
		resultBuilder.WriteString("<div class='answers'>")
		for i, q := range quiz.Questions {
			userAns := answers[i]
			correctAns := q.Correct
			if userAns == correctAns {
				correctCount++
			}
			userColor := "#dc3545"
			if userAns == correctAns {
				userColor = "#0d6efd"
			}
			resultBuilder.WriteString(fmt.Sprintf(`
				<div class="answer-item mb-3 p-3 border rounded">
					<strong>Вопрос %d</strong><br>
					%s<br>
					<span style="color: %s;">Ваш ответ: %s</span><br>
					<span class="text-success">Правильный: %s</span>
				</div>`, i+1, q.Text, userColor,
				optionText(q.Options, userAns),
				optionText(q.Options, correctAns),
			))
		}
		resultBuilder.WriteString("</div>")

		writeLog(username, correctCount, len(quiz.Questions), duration)
		delete(quizzes, session)

		c.HTML(http.StatusOK, "result.html", gin.H{
			"Username":    username,
			"Correct":     correctCount,
			"Total":       len(quiz.Questions),
			"Time":        int(duration.Minutes()),
			"AnswersHTML": template.HTML(resultBuilder.String()),
		})
	})

	r.Run(":8080")
}

func optionText(options []string, idx int) string {
	if idx > 0 && idx <= len(options) {
		return options[idx-1]
	}
	return "<em>не выбрано</em>"
}
