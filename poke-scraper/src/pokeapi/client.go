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

func (c *Client) fetchPokemon(url string,
	errChan chan<- error,
	resultChan chan<- PokemonResponse,
	waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	c.sugar.Infof("Fetching PokemonResponse %s", url)
	response, err := c.client.Get(url)
	if err != nil {
		errChan <- err
		return
	}
	var pokemon PokemonResponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&pokemon)
	if err != nil {
		errChan <- err
		return
	}
	resultChan <- pokemon
	err = response.Body.Close()
	if err != nil {
		errChan <- err
		return
	}
}

func (c *Client) ListPokemons(limit, offset int32) ([]PokemonResponse, error) {
	url := fmt.Sprintf("%s/pokemon?limit=%d&offset=%d", c.baseUrl, limit, offset)
	response, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	var result PokemonListResult
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&result)
	if err != nil {
		return nil, err
	}
	err = response.Body.Close()
	if err != nil {
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
	err = errors.Join(errs...)
	if err != nil {
		return nil, err
	}
	var results []PokemonResponse
	for pokemon := range resultChan {
		results = append(results, pokemon)
	}
	return results, nil
}

func (c *Client) GetPokemonGeneration(species PokemonResponseSpecies) (int32, error) {
	response, err := c.client.Get(species.Url)
	if err != nil {
		return 0, err
	}
	var pokemonSpecies PokemonSpecies
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&pokemonSpecies)
	if err != nil {
		return 0, err
	}
	err = response.Body.Close()
	if err != nil {
		return 0, err
	}
	response, err = c.client.Get(pokemonSpecies.Generation.Url)
	if err != nil {
		return 0, err
	}
	var generation PokemonGeneration
	decoder = json.NewDecoder(response.Body)
	err = decoder.Decode(&generation)
	if err != nil {
		return 0, err
	}
	err = response.Body.Close()
	if err != nil {
		return 0, err
	}
	return generation.Id, nil
}
