package pokeapi

type PokemonListResultEntry struct {
	Url string `json:"url"`
}

type PokemonListResult struct {
	Results []PokemonListResultEntry `json:"results"`
}

type PokemonTypeEntry struct {
	Name string `json:"name"`
}

type PokemonType struct {
	Slot int32            `json:"slot"`
	Type PokemonTypeEntry `json:"type"`
}

type PokemonResponse struct {
	Name   string        `json:"name"`
	Weight int32         `json:"weight"`
	Height int32         `json:"height"`
	Types  []PokemonType `json:"types"`
}
