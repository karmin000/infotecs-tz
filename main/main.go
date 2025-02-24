package main

import (
	"errors"
	"infotecs-tz/logger/slogpretty"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth_gin"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Wallet struct {
	Address string  `gorm:"primaryKey" json:"address"`
	Balance float64 `json:"balance"`
}

type Transaction struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

var db *gorm.DB
var logger *slog.Logger
var addressRegex = regexp.MustCompile(`^[a-f0-9]{64}$`) // HEX-адрес длиной 64 символа

func initDB() {
	logger = setupPrettySlog()
	var err error
	os.MkdirAll("storage", os.ModePerm)
	db, err = gorm.Open(sqlite.Open("storage/storage.db"), &gorm.Config{})
	if err != nil {
		logger.Error("Failed to connect to database", slog.String("error", err.Error()))
		panic("Failed to connect to database")
	}
	db.AutoMigrate(&Wallet{}, &Transaction{})
	seedWallets()
}

func seedWallets() {
	var count int64
	db.Model(&Wallet{}).Count(&count)
	if count == 0 {
		for i := 0; i < 10; i++ {
			wallet := Wallet{Address: generateWalletAddress(), Balance: 100.0}
			db.Create(&wallet)
		}
	}
}

func generateWalletAddress() string {
	letters := "abcdef0123456789"
	for {
		addr := make([]byte, 64)
		for i := range addr {
			addr[i] = letters[rand.Intn(len(letters))]
		}
		address := string(addr)

		wallet := Wallet{Address: address, Balance: 100.0}
		if err := db.Create(&wallet).Error; err == nil {
			return address
		}
	}
}

func validateAddress(addr string) bool {
	return addressRegex.MatchString(addr)
}

func sendTransaction(c *gin.Context) {
	var tx Transaction
	if err := c.ShouldBindJSON(&tx); err != nil {
		logger.Warn("Invalid request body", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if !validateAddress(tx.From) || !validateAddress(tx.To) {
		logger.Warn("Invalid wallet address format", slog.String("from", tx.From), slog.String("to", tx.To))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet address format"})
		return
	}

	if tx.From == tx.To {
		logger.Warn("Invalid wallet address format. Cannot send funds to the same wallet", slog.String("from", tx.From), slog.String("to", tx.To))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot send funds to the same wallet"})
		return
	}

	if tx.Amount <= 0 {
		logger.Warn("Invalid transaction amount", slog.Float64("amount", tx.Amount))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be greater than zero"})
		return
	}

	tx.Timestamp = time.Now()

	txErr := db.Transaction(func(txDB *gorm.DB) error {
		var sender, receiver Wallet
		if err := txDB.First(&sender, "address = ?", tx.From).Error; err != nil {
			logger.Warn("Sender wallet not found", slog.String("address", tx.From))
			return errors.New("Sender wallet not found")
		}
		if sender.Balance < tx.Amount {
			logger.Warn("Insufficient funds", slog.String("address", tx.From), slog.Float64("balance", sender.Balance), slog.Float64("amount", tx.Amount))
			return errors.New("Insufficient funds")
		}
		if err := txDB.First(&receiver, "address = ?", tx.To).Error; err != nil {
			logger.Warn("Receiver wallet not found", slog.String("address", tx.To))
			return errors.New("Receiver wallet not found")
		}
		if err := txDB.Model(&Wallet{}).Where("address = ?", sender.Address).
			Update("balance", gorm.Expr("balance - ?", tx.Amount)).Error; err != nil {
			logger.Error("Failed to update sender wallet", slog.String("address", sender.Address), slog.String("error", err.Error()))
			return err
		}
		if err := txDB.Model(&Wallet{}).Where("address = ?", receiver.Address).
			Update("balance", gorm.Expr("balance + ?", tx.Amount)).Error; err != nil {
			logger.Error("Failed to update receiver wallet", slog.String("address", receiver.Address), slog.String("error", err.Error()))
			return err
		}
		if err := txDB.Create(&tx).Error; err != nil {
			logger.Error("Failed to record transaction", slog.String("error", err.Error()))
			return err
		}
		return nil
	})

	if txErr != nil {
		logger.Warn("Transaction failed",
			slog.String("from", maskAddress(tx.From)),
			slog.String("to", maskAddress(tx.To)),
			slog.Float64("amount", tx.Amount),
			slog.String("error", txErr.Error()),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": txErr.Error()})
		return
	}

	logger.Info("Transaction completed successfully",
		slog.String("from", maskAddress(tx.From)),
		slog.String("to", maskAddress(tx.To)),
		slog.Float64("amount", tx.Amount))
	c.JSON(http.StatusOK, tx)
}

func maskAddress(addr string) string {
	if len(addr) < 10 {
		return "******"
	}
	return addr[:5] + "..." + addr[len(addr)-5:]
}

func getLastTransactions(c *gin.Context) {
	count := 10
	countParam := c.Query("count")
	if countParam != "" {
		if parsedCount, err := strconv.Atoi(countParam); err == nil && parsedCount > 0 {
			count = parsedCount
			if count > 1000 {
				count = 1000
			}
		} else {
			logger.Warn("Invalid count parameter", slog.String("count", countParam))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid count parameter"})
			return
		}
	}

	var transactions []Transaction
	if err := db.Order("timestamp desc").Limit(count).Find(&transactions).Error; err != nil {
		logger.Error("Failed to retrieve transactions", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve transactions"})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

func getBalance(c *gin.Context) {
	address := c.Param("address")
	if !validateAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet address format"})
		return
	}

	var wallet Wallet
	if err := db.First(&wallet, "address = ?", address).Error; err != nil {
		logger.Warn("Wallet not found", slog.String("address", address))
		c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
		return
	}
	c.JSON(http.StatusOK, wallet)
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	initDB()
	r := gin.Default()

	limiter := tollbooth.NewLimiter(1, nil) // 1 запрос в секунду на IP
	r.POST("/api/send", tollbooth_gin.LimitHandler(limiter), sendTransaction)
	r.GET("/api/transactions", getLastTransactions)
	r.GET("/api/wallet/:address/balance", getBalance)

	logger.Info("Server is running", slog.String("port", "8080"))
	r.Run(":8080")
}
