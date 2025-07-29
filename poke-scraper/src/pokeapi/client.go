package pokeapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"go.uber.org/zap"
)

type Client struct {
	baseUrl string
	client  *http.Client
	sugar   *zap.SugaredLogger
}

func NewClient(sugar *zap.SugaredLogger) *Client {
	return &Client{
		baseUrl: "https://pokeapi.co/api/v2/",
		client:  &http.Client{},
		sugar:   sugar,
	}
}

func (c *Client) getAndDecode(url string, target any) error {
	resp, err := c.client.Get(url)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := resp.Body.Close(); err != nil {
		return err
	}
	return nil
}

func (c *Client) fetchPokemon(url string,
	errChan chan<- error,
	resultChan chan<- PokemonResponse,
	waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	c.sugar.Infof("Fetching PokemonResponse %s", url)
	var pokemon PokemonResponse
	if err := c.getAndDecode(url, &pokemon); err != nil {
		errChan <- err
		return
	}
	resultChan <- pokemon
}

func (c *Client) ListPokemons(limit, offset int32) ([]PokemonResponse, error) {
	url := fmt.Sprintf("%s/pokemon?limit=%d&offset=%d", c.baseUrl, limit, offset)
	var result PokemonListResult
	if err := c.getAndDecode(url, &result); err != nil {
		return nil, err
	}
	var waitGroup sync.WaitGroup
	errChan := make(chan error, len(result.Results))
	resultChan := make(chan PokemonResponse, len(result.Results))
	for _, entry := range result.Results {
		waitGroup.Add(1)
		go c.fetchPokemon(entry.Url, errChan, resultChan, &waitGroup)
	}
	waitGroup.Wait()
	close(errChan)
	close(resultChan)
	var errs []error
	for e := range errChan {
		if e != nil {
			errs = append(errs, e)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	var results []PokemonResponse
	for pokemon := range resultChan {
		results = append(results, pokemon)
	}
	return results, nil
}

func (c *Client) GetPokemonGeneration(species PokemonResponseSpecies) (int32, error) {
	var pokemonSpecies PokemonSpecies
	if err := c.getAndDecode(species.Url, &pokemonSpecies); err != nil {
		return 0, err
	}
	var generation PokemonGeneration
	if err := c.getAndDecode(pokemonSpecies.Generation.Url, &generation); err != nil {
		return 0, err
	}
	return generation.Id, nil
}
