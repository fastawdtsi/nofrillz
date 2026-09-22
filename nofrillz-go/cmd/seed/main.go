package main

import (
	"context"
	"errors"
	"fmt"
	mathrand "math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	mysqlerr "github.com/go-sql-driver/mysql"

	"nofrillz/internal/config"
	"nofrillz/internal/db"
	"nofrillz/internal/follows"
	"nofrillz/internal/logging"
	"nofrillz/internal/posts"
	"nofrillz/internal/snowid"
	"nofrillz/internal/users"
)

const (
	defaultPassword  = "password"
	minFollows       = 10
	maxFollows       = 100
	maxPostsPerUser  = 100
	progressLogEvery = 25
	maxUserRetries   = 50
)

var firstNames = []string{
	"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda", "William", "Elizabeth",
	"David", "Barbara", "Richard", "Susan", "Joseph", "Jessica", "Thomas", "Sarah", "Charles", "Karen",
	"Christopher", "Nancy", "Daniel", "Lisa", "Matthew", "Betty", "Anthony", "Margaret", "Mark", "Sandra",
	"Donald", "Ashley", "Steven", "Kimberly", "Paul", "Emily", "Andrew", "Donna", "Joshua", "Michelle",
	"Kenneth", "Dorothy", "Kevin", "Carol", "Brian", "Amanda", "George", "Melissa", "Edward", "Deborah",
	"Ronald", "Stephanie", "Timothy", "Rebecca", "Jason", "Sharon", "Jeffrey", "Laura", "Ryan", "Cynthia",
	"Jacob", "Kathleen", "Gary", "Amy", "Nicholas", "Shirley", "Eric", "Angela", "Jonathan", "Helen",
	"Stephen", "Anna", "Larry", "Brenda", "Justin", "Pamela", "Scott", "Nicole", "Brandon", "Samantha",
	"Benjamin", "Katherine", "Samuel", "Emma", "Frank", "Ruth", "Gregory", "Christine", "Raymond", "Catherine",
	"Alexander", "Debra", "Patrick", "Rachel", "Jack", "Carolyn", "Dennis", "Janet", "Jerry", "Virginia",
}

var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez",
	"Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin",
	"Lee", "Perez", "Thompson", "White", "Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson",
	"Walker", "Young", "Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores",
	"Green", "Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell", "Mitchell", "Carter", "Roberts",
	"Gomez", "Phillips", "Evans", "Turner", "Diaz", "Parker", "Cruz", "Edwards", "Collins", "Reyes",
	"Stewart", "Morris", "Morales", "Murphy", "Cook", "Rogers", "Gutierrez", "Ortiz", "Morgan", "Cooper",
	"Peterson", "Bailey", "Reed", "Kelly", "Howard", "Ramos", "Kim", "Cox", "Ward", "Richardson",
	"Watson", "Brooks", "Chavez", "Wood", "James", "Bennett", "Gray", "Mendoza", "Ruiz", "Hughes",
	"Price", "Alvarez", "Castillo", "Sanders", "Patel", "Myers", "Long", "Ross", "Foster", "Jimenez",
}

var loremWords = []string{
	"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit", "sed", "do",
	"eiusmod", "tempor", "incididunt", "ut", "labore", "et", "dolore", "magna", "aliqua", "ut",
	"enim", "ad", "minim", "veniam", "quis", "nostrud", "exercitation", "ullamco", "laboris", "nisi",
	"ut", "aliquip", "ex", "ea", "commodo", "consequat", "duis", "aute", "irure", "dolor",
	"in", "reprehenderit", "in", "voluptate", "velit", "esse", "cillum", "dolore", "eu", "fugiat",
	"nulla", "pariatur", "excepteur", "sint", "occaecat", "cupidatat", "non", "proident", "sunt", "in",
	"culpa", "qui", "officia", "deserunt", "mollit", "anim", "id", "est", "laborum",
}

type seededUser struct {
	ID       uint64
	Username string
}

func main() {
	if len(os.Args) != 3 {
		panic(fmt.Sprintf("usage: %s <config.yaml> <user_count>", os.Args[0]))
	}

	configPath := os.Args[1]
	userCount, err := strconv.Atoi(os.Args[2])
	if err != nil || userCount <= 0 {
		panic("user_count must be a positive integer")
	}

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		panic(fmt.Errorf("error in config.NewConfig: %w", err))
	}

	logger, level, levelErr := logging.NewLogger("seed", cfg.LogConfig().Level)
	if levelErr != nil {
		logger.Warn().Str("configured_level", cfg.LogConfig().Level).Msg("invalid log level configured, defaulting to info")
	}
	logger.Info().Int("user_count", userCount).Str("level", level.String()).Msg("starting seed")

	mysql, err := db.NewMySQL(cfg.MySQLConfig())
	if err != nil {
		panic(err)
	}
	defer mysql.Close()

	idGenerator, err := snowid.NewDefault(cfg.IDGeneratorConfig())
	if err != nil {
		panic(err)
	}

	usersService := users.NewService(users.NewRepository(mysql.DB))
	followsService := follows.NewService(follows.NewRepository(mysql.DB))

	ctx := context.Background()
	rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	start := time.Now()
	createdUsers := 0
	createdPosts := 0
	createdFollows := 0

	passwordHash, passwordSalt, err := users.GeneratePasswordHashAndSalt(defaultPassword)
	if err != nil {
		panic(fmt.Errorf("hash default password failed: %w", err))
	}

	seededUsers := make([]seededUser, 0, userCount)
	usernameCounters := map[string]int{}

	for i := 0; i < userCount; i++ {
		firstName := firstNames[rng.Intn(len(firstNames))]
		lastName := lastNames[rng.Intn(len(lastNames))]
		var createdUser *users.User
		for attempt := 0; attempt < maxUserRetries; attempt++ {
			username := buildUsername(firstName, lastName, usernameCounters)
			email := fmt.Sprintf("%s_%d@example.com", username, i+1)

			u := &users.User{
				ID:          idGenerator.MustNext(),
				Email:       email,
				Username:    username,
				FirstName:   firstName,
				LastName:    lastName,
				AccountType: users.AccountTypeHuman,

				PasswordHash: passwordHash,
				PasswordSalt: passwordSalt,
			}

			err := usersService.Create(ctx, u)
			if err == nil {
				createdUser = u
				break
			}
			if isDuplicateKeyError(err) {
				logger.Warn().
					Int("seed_index", i).
					Int("attempt", attempt+1).
					Str("username", u.Username).
					Str("email", u.Email).
					Msg("duplicate seeded user, retrying")
				continue
			}

			panic(fmt.Errorf("create user %d failed: %w", i, err))
		}

		if createdUser == nil {
			panic(fmt.Errorf("create user %d failed after %d retries", i, maxUserRetries))
		}

		seededUsers = append(seededUsers, seededUser{ID: createdUser.ID, Username: createdUser.Username})
		createdUsers++
		if createdUsers%progressLogEvery == 0 {
			logger.Info().Int("users", createdUsers).Int("posts", createdPosts).Int("follows", createdFollows).Msg("seed progress")
		}
	}

	if len(seededUsers) == 0 {
		logger.Info().Msg("seed completed with no users")
		return
	}

	hubUserID := seededUsers[0].ID
	for _, u := range seededUsers[1:] {
		if _, err := followsService.Follow(ctx, hubUserID, u.ID); err != nil {
			panic(fmt.Errorf("hub follow user %d failed: %w", u.ID, err))
		}
		createdFollows++

		if _, err := followsService.Follow(ctx, u.ID, hubUserID); err != nil {
			panic(fmt.Errorf("user %d follow hub failed: %w", u.ID, err))
		}
		createdFollows++
	}

	for _, u := range seededUsers {
		if len(seededUsers) <= 1 {
			break
		}

		if u.ID == hubUserID {
			continue
		}

		availableOthers := len(seededUsers) - 1
		targetMin := minInt(minFollows, availableOthers)
		targetMax := minInt(maxFollows, availableOthers)
		targetCount := targetMin
		if targetMax > targetMin {
			targetCount = targetMin + rng.Intn(targetMax-targetMin+1)
		}

		selected := map[uint64]bool{hubUserID: true}
		for len(selected) < targetCount {
			target := seededUsers[rng.Intn(len(seededUsers))].ID
			if target == u.ID {
				continue
			}
			selected[target] = true
		}

		for followingID := range selected {
			if _, err := followsService.Follow(ctx, u.ID, followingID); err != nil {
				panic(fmt.Errorf("random follow %d->%d failed: %w", u.ID, followingID, err))
			}
			createdFollows++
		}
	}

	now := time.Now().UTC()
	oneYearAgo := now.AddDate(-1, 0, 0)
	for _, u := range seededUsers {
		postCount := rng.Intn(maxPostsPerUser + 1)
		for j := 0; j < postCount; j++ {
			createdAt := randomTimeBetween(rng, oneYearAgo, now)
			body := randomLoremSentence(rng, 5, 100)

			_, err := mysql.DB.Exec(
				"insert into posts (id,user_id,body,source,created,updated) values (?,?,?,?,?,?)",
				idGenerator.MustNext(),
				u.ID,
				body,
				posts.SourceHuman,
				createdAt,
				createdAt,
			)
			if err != nil {
				panic(fmt.Errorf("create post %d for user %d failed: %w", j, u.ID, err))
			}
			createdPosts++
		}
	}

	logger.Info().
		Int("users", createdUsers).
		Int("posts", createdPosts).
		Int("follows", createdFollows).
		Dur("elapsed", time.Since(start).Round(time.Millisecond)).
		Msg("seed completed")
}

func buildUsername(firstName string, lastName string, counters map[string]int) string {
	base := strings.ToLower(firstName + lastName)
	count := counters[base]
	counters[base] = count + 1
	if count == 0 {
		return base
	}

	return base + strconv.Itoa(count+1)
}

func isDuplicateKeyError(err error) bool {
	var mysqlError *mysqlerr.MySQLError
	return errors.As(err, &mysqlError) && mysqlError.Number == 1062
}

func randomTimeBetween(rng *mathrand.Rand, start time.Time, end time.Time) time.Time {
	if !end.After(start) {
		return end
	}

	delta := end.UnixNano() - start.UnixNano()
	offset := rng.Int63n(delta + 1)
	return time.Unix(0, start.UnixNano()+offset).UTC()
}

func randomLoremSentence(rng *mathrand.Rand, minWords int, maxWords int) string {
	if minWords <= 0 {
		minWords = 1
	}
	if maxWords < minWords {
		maxWords = minWords
	}

	count := minWords
	if maxWords > minWords {
		count = minWords + rng.Intn(maxWords-minWords+1)
	}

	words := make([]string, 0, count)
	for i := 0; i < count; i++ {
		words = append(words, loremWords[rng.Intn(len(loremWords))])
	}

	if len(words) > 0 {
		words[0] = strings.Title(words[0])
	}

	return strings.Join(words, " ") + "."
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}

	return b
}
