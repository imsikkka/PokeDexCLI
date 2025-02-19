package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"PokeDexCli/internal/pokecache"
)

// API response structs
type locationResponse struct {
	Results []struct {
		Name string `json:"name"`
	} `json:"results"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
}

type locationDetailsResponse struct {
	PokemonEncounters []struct {
		Pokemon struct {
			Name string `json:"name"`
		} `json:"pokemon"`
	} `json:"pokemon_encounters"`
}

type pokemonResponse struct {
	Name           string `json:"name"`
	BaseExperience int    `json:"base_experience"`
	Height         int    `json:"height"`
	Weight         int    `json:"weight"`
	Stats          []struct {
		BaseStat int `json:"base_stat"`
		Stat     struct {
			Name string `json:"name"`
		} `json:"stat"`
	} `json:"stats"`
	Types []struct {
		Type struct {
			Name string `json:"name"`
		} `json:"type"`
	} `json:"types"`
}

// Configuration struct
type config struct {
	cache    *pokecache.Cache
	next     string
	previous string
	pokedex  map[string]pokemonResponse
	commands map[string]cliCommand
}

// CLI command struct
type cliCommand struct {
	name        string
	description string
	callback    func(*config, []string) error
}

// Fetch data with caching
func fetchData(url string, cfg *config, target interface{}) error {
	if cachedData, found := cfg.cache.Get(url); found {
		fmt.Println("(Cache hit)")
		return json.Unmarshal(cachedData, target)
	}

	fmt.Println("(Fetching from API)")
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	cfg.cache.Add(url, body)
	return json.Unmarshal(body, target)
}

// Command implementations
func commandExit(cfg *config, args []string) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(cfg *config, args []string) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	for _, cmd := range cfg.commands {
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}
	return nil
}

func commandMap(cfg *config, args []string) error {
	apiURL := "https://pokeapi.co/api/v2/location-area/"
	if cfg.next != "" {
		apiURL = cfg.next
	}

	var data locationResponse
	err := fetchData(apiURL, cfg, &data)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	for _, location := range data.Results {
		fmt.Println(location.Name)
	}

	cfg.next = data.Next
	cfg.previous = data.Previous
	return nil
}

func commandMapBack(cfg *config, args []string) error {
	if cfg.previous == "" {
		fmt.Println("You're on the first page.")
		return nil
	}

	var data locationResponse
	err := fetchData(cfg.previous, cfg, &data)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	for _, location := range data.Results {
		fmt.Println(location.Name)
	}

	cfg.next = data.Next
	cfg.previous = data.Previous
	return nil
}

func commandExplore(cfg *config, args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: explore <location>")
		return nil
	}

	url := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%s/", args[0])
	var data locationDetailsResponse
	err := fetchData(url, cfg, &data)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	fmt.Println("Found Pokémon:")
	for _, pokemon := range data.PokemonEncounters {
		fmt.Printf(" - %s\n", pokemon.Pokemon.Name)
	}

	return nil
}

// ✅ Fixed: Catches Pokémon correctly
func commandCatch(cfg *config, args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: catch <pokemon>")
		return nil
	}

	pokemonName := strings.ToLower(args[0]) // Convert to lowercase
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s/", pokemonName)

	var data pokemonResponse
	err := fetchData(url, cfg, &data)
	if err != nil {
		fmt.Println("Error fetching Pokémon:", err)
		return err
	}

	fmt.Printf("Throwing a Pokeball at %s...\n", data.Name)
	rand.Seed(time.Now().UnixNano())
	catchChance := 100 - data.BaseExperience

	if rand.Intn(100) < catchChance {
		fmt.Printf("%s was caught!\n", data.Name)
		cfg.pokedex[pokemonName] = data // Store in lowercase
		fmt.Println("You may now inspect it with the inspect command.")
	} else {
		fmt.Printf("%s escaped!\n", data.Name)
	}

	return nil
}

// ✅ Fixed: Now recognizes caught Pokémon properly
func commandInspect(cfg *config, args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: inspect <pokemon>")
		return nil
	}

	pokemonName := strings.ToLower(args[0]) // Convert input to lowercase

	pokemon, exists := cfg.pokedex[pokemonName]
	if !exists {
		fmt.Println("You have not caught that Pokémon.")
		return nil
	}

	fmt.Printf("Name: %s\nHeight: %d\nWeight: %d\n", pokemon.Name, pokemon.Height, pokemon.Weight)
	fmt.Println("Stats:")
	for _, stat := range pokemon.Stats {
		fmt.Printf("  - %s: %d\n", stat.Stat.Name, stat.BaseStat)
	}
	fmt.Println("Types:")
	for _, typ := range pokemon.Types {
		fmt.Printf("  - %s\n", typ.Type.Name)
	}

	return nil
}

// ✅ Fixed: Lists all caught Pokémon
func commandPokedex(cfg *config, args []string) error {
	if len(cfg.pokedex) == 0 {
		fmt.Println("Your Pokedex is empty. Catch some Pokémon first!")
		return nil
	}

	fmt.Println("Your Pokedex:")
	for name := range cfg.pokedex {
		fmt.Printf(" - %s\n", name) // Names are stored lowercase
	}

	return nil
}

func main() {
	cfg := &config{
		cache:    pokecache.NewCache(10 * time.Second),
		pokedex:  make(map[string]pokemonResponse),
		commands: make(map[string]cliCommand),
	}

	// ✅ Register All Commands
	cfg.commands["exit"] = cliCommand{"exit", "Exit the Pokedex", commandExit}
	cfg.commands["help"] = cliCommand{"help", "Displays help", commandHelp}
	cfg.commands["map"] = cliCommand{"map", "Show locations", commandMap}
	cfg.commands["mapb"] = cliCommand{"mapb", "Go back", commandMapBack}
	cfg.commands["explore"] = cliCommand{"explore", "See Pokémon in an area", commandExplore}
	cfg.commands["catch"] = cliCommand{"catch", "Catch a Pokémon", commandCatch}
	cfg.commands["inspect"] = cliCommand{"inspect", "Inspect a caught Pokémon", commandInspect}
	cfg.commands["pokedex"] = cliCommand{"pokedex", "View caught Pokémon", commandPokedex}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Pokedex > ")
		scanner.Scan()
		words := strings.Fields(strings.ToLower(scanner.Text()))
		if len(words) == 0 {
			continue
		}
		if cmd, found := cfg.commands[words[0]]; found {
			cmd.callback(cfg, words[1:])
		} else {
			fmt.Println("Unknown command")
		}
	}
}
