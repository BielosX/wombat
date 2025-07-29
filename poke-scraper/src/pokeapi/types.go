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

type PokemonStatEntry struct {
	BaseStat int32              `json:"base_stat"`
	Stat     PokemonStatDetails `json:"stat"`
}

type PokemonStatDetails struct {
	Name string `json:"name"`
}

type PokemonResponseSpecies struct {
	Url string `json:"url"`
}

type PokemonResponse struct {
	Id      int32                  `json:"id"`
	Name    string                 `json:"name"`
	Weight  int32                  `json:"weight"`
	Height  int32                  `json:"height"`
	Types   []PokemonType          `json:"types"`
	Stats   []PokemonStatEntry     `json:"stats"`
	Species PokemonResponseSpecies `json:"species"`
}

type PokemonSpecies struct {
	Generation PokemonSpeciesGeneration `json:"generation"`
}

type PokemonSpeciesGeneration struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

type PokemonGeneration struct {
	Id int32 `json:"id"`
}
