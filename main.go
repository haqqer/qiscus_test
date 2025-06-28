package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var (
	APP_ID            = "tqxyb-ivaigtav3oku4cj"
	SECRET_KEY        = "6df96f4525cf12f2bc315d4c1eb98d5e"
	CHANNEL_ID        = "131203"
	BASE_URL          = "https://omnichannel.qiscus.com"
	TASK_QUEUE_KEY    = "TASK_QUEUE"
	RESOLVE_QUEUE_KEY = "RESOLVE_QUEUE"
	REDIS_HOST        = "127.0.0.1:6379"
	REDIS_PASS        = ""
	MAX_CUSTOMER      = 5
)

type ListAvailableAgents struct {
	Data struct {
		Agents []Agent `json:"agents"`
	} `json:"data"`
	Meta struct {
		After      interface{} `json:"after"`
		Before     interface{} `json:"before"`
		PerPage    int         `json:"per_page"`
		TotalCount interface{} `json:"total_count"`
	} `json:"meta"`
	Status int `json:"status"`
}

type Agent struct {
	AvatarURL            string      `json:"avatar_url"`
	CreatedAt            string      `json:"created_at"`
	CurrentCustomerCount int         `json:"current_customer_count"`
	Email                string      `json:"email"`
	ForceOffline         bool        `json:"force_offline"`
	ID                   int         `json:"id"`
	IsAvailable          bool        `json:"is_available"`
	IsReqOtpReset        interface{} `json:"is_req_otp_reset"`
	LastLogin            time.Time   `json:"last_login"`
	Name                 string      `json:"name"`
	SdkEmail             string      `json:"sdk_email"`
	SdkKey               string      `json:"sdk_key"`
	Type                 int         `json:"type"`
	TypeAsString         string      `json:"type_as_string"`
	UserChannels         []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"user_channels"`
	UserRoles []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"user_roles"`
}

var selectedAgent Agent

type RedisQueue struct {
	client *redis.Client
	key    string
}

// NewRedisQueue creates a new Redis FIFO queue
func NewRedisQueue(addr, password string, db int, queueKey string) *RedisQueue {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisQueue{
		client: rdb,
		key:    queueKey,
	}
}
func ResponseSuccess(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"error":  false,
		"status": http.StatusOK,
		"data":   payload,
	}
	json.NewEncoder(w).Encode(response)
}

func ResponseError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	response := map[string]interface{}{
		"status":  status,
		"message": message,
		"error":   true,
	}
	json.NewEncoder(w).Encode(response)
}

func GetAllAgents() (*ListAvailableAgents, error) {
	req, err := http.NewRequest("GET", BASE_URL+"/api/v2/admin/agents", nil)
	if err != nil {
		log.Printf("Error creating request: %v\n", err)
		return nil, err
	}
	req.Header.Set("Qiscus-Secret-Key", SECRET_KEY)
	req.Header.Set("Qiscus-App-Id", APP_ID)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close() // Ensure the response body is closed

	payload := &ListAvailableAgents{}
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func AssignAgent(room_id string, agent_id int) (map[string]interface{}, error) {
	formData := url.Values{}
	formData.Set("room_id", room_id)
	formData.Set("agent_id", strconv.Itoa(agent_id))

	req, err := http.NewRequest("POST", BASE_URL+"/api/v1/admin/service/assign_agent", strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("Error creating request: %v\n", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Qiscus-Secret-Key", SECRET_KEY)
	req.Header.Set("Qiscus-App-Id", APP_ID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close() // Ensure the response body is closed

	payload := map[string]interface{}{}
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}
	return payload, nil

}

func GetAvailableAgent(room_id string) (*ListAvailableAgents, error) {
	params := url.Values{}
	params.Add("room_id", room_id)

	req, err := http.NewRequest("GET", BASE_URL+"/api/v2/admin/service/available_agents?"+params.Encode(), nil)
	if err != nil {
		log.Printf("Error creating request: %v\n", err)
		return nil, err
	}
	req.Header.Set("Qiscus-Secret-Key", SECRET_KEY)
	req.Header.Set("Qiscus-App-Id", APP_ID)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close() // Ensure the response body is closed

	payload := &ListAvailableAgents{}
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func MarkAsResolvedAdmin(room_id string) (map[string]interface{}, error) {
	formData := url.Values{}
	formData.Set("room_id", room_id)

	req, err := http.NewRequest("POST", BASE_URL+"/api/v1/admin/service/mark_as_resolved", strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("Error creating request: %v\n", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Qiscus-Secret-Key", SECRET_KEY)
	req.Header.Set("Qiscus-App-Id", APP_ID)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close() // Ensure the response body is closed

	payload := map[string]interface{}{}
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func Serve() {
	ctx := context.Background()
	queue := NewRedisQueue(REDIS_HOST, REDIS_PASS, 0, TASK_QUEUE_KEY)
	defer queue.client.Close()

	ping, err := queue.client.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}
	log.Println(ping)

	go func(ctx context.Context) {
		log.Println("Assignment Worker is running...")
		var selectedAgent Agent
		for {
			result := queue.client.LIndex(ctx, TASK_QUEUE_KEY, 0)

			if result.Err() == redis.Nil {
				log.Println("QUEUE TASK EMPTY!")
			} else {
				roomID := result.Val()
				resp, err := GetAvailableAgent(roomID)
				if err != nil {
					log.Println("error GetAllAgnet : ", err.Error())
					continue
				}
				if len(resp.Data.Agents) > 0 {
					selectedAgent = resp.Data.Agents[0]
					for _, agent := range resp.Data.Agents {
						fmt.Printf("PERBANDINGAN agent vs selectedAgent : %d vs %d\n", agent.CurrentCustomerCount, selectedAgent.CurrentCustomerCount)
						if agent.IsAvailable && agent.CurrentCustomerCount <= selectedAgent.CurrentCustomerCount && agent.CurrentCustomerCount <= MAX_CUSTOMER {
							selectedAgent = agent
						}
					}
				} else {
					log.Println("Agents empty")
					continue
				}
				log.Println("ASSIGN agent_ID " + selectedAgent.Email + " room_ID: " + roomID)
				respAssignAgent, err := AssignAgent(roomID, selectedAgent.ID)
				if err != nil {
					log.Println("error AssignAgent : ", err.Error())
					time.Sleep(5 * time.Second)
					continue
				}
				log.Println(respAssignAgent)
				err = queue.client.LPop(ctx, TASK_QUEUE_KEY).Err()
				if err != nil {
					log.Println("error process LPop Task Queue")
				}
				err = queue.client.RPush(ctx, RESOLVE_QUEUE_KEY, roomID).Err()
				if err != nil {
					log.Println("error process RPush Resolve Queue")
				}
			}
			time.Sleep(5 * time.Second)
		}
	}(ctx)

	go func(ctx context.Context) {
		log.Println("Resolver Worker is running...")
		for {
			resultResolve := queue.client.LIndex(ctx, RESOLVE_QUEUE_KEY, 0)

			if resultResolve.Err() == redis.Nil {
				log.Println("QUEUE RESOLVE EMPTY!")
			} else {
				roomIDResolve := resultResolve.Val()
				log.Println("RESOLVING room_ID: " + roomIDResolve)
				resp, err := MarkAsResolvedAdmin(roomIDResolve)
				if err != nil {
					log.Println("error MarkAsResolvedAdmin : ", err.Error())
					time.Sleep(5 * time.Second)
					continue
				}
				log.Println("successfully process", resp)
				err = queue.client.LPop(ctx, RESOLVE_QUEUE_KEY).Err()
				if err != nil {
					log.Println("error process LPop Resolve Queue")
				}
			}
			time.Sleep(3 * time.Second)
		}
	}(ctx)

	router := http.NewServeMux()
	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		ResponseSuccess(w, map[string]string{"status": "OK"})
	})
	router.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {
		payload := map[string]interface{}{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			log.Println("Error decoder JSON:", err)
		}
		err = queue.client.RPush(r.Context(), TASK_QUEUE_KEY, payload["room_id"].(string)).Err()
		if err != nil {
			log.Println("error push to queue", err.Error())
		}
		ResponseSuccess(w, payload)
	})
	// Create a new server
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start the server in a goroutine
	go func() {
		fmt.Println("Server is starting on port 8080...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", server.Addr, err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v\n", err)
	}

	log.Println("Server exiting")
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %s", err)
	}
	APP_ID = os.Getenv("APP_ID")
	SECRET_KEY = os.Getenv("SECRET_KEY")
	CHANNEL_ID = os.Getenv("CHANNEL_ID")
	BASE_URL = os.Getenv("BASE_URL")
	TASK_QUEUE_KEY = os.Getenv("TASK_QUEUE")
	RESOLVE_QUEUE_KEY = os.Getenv("RESOLVE_QUEUE")

	var argsRaw = os.Args
	if len(argsRaw) <= 1 {
		fmt.Println("use `run` or `migrate` arguments")
		os.Exit(1)
	}
	arg := argsRaw[1]
	switch arg {
	case "serve":
		Serve()
	}
}
