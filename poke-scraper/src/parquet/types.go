package parquet

import (
	"errors"
	"fmt"

	"github.com/BielosX/wombat/poke-scraper/src/pokeapi"
)

type Pokemon struct {
	PokemonId          int32  `parquet:"name=pokemonId, type=INT32"`
	Name               string `parquet:"name=name, type=BYTE_ARRAY, convertedtype=UTF8"`
	Weight             int32  `parquet:"name=weight, type=INT32"`
	Height             int32  `parquet:"name=height, type=INT32"`
	Type               string `parquet:"name=type, type=BYTE_ARRAY, convertedtype=UTF8"`
	BaseHp             int32  `parquet:"name=baseHp, type=INT32"`
	BaseAttack         int32  `parquet:"name=baseAttack, type=INT32"`
	BaseDefense        int32  `parquet:"name=baseDefense, type=INT32"`
	BaseSpecialAttack  int32  `parquet:"name=baseSpecialAttack, type=INT32"`
	BaseSpecialDefense int32  `parquet:"name=baseSpecialDefense, type=INT32"`
	BaseSpeed          int32  `parquet:"name=baseSpeed, type=INT32"`
	Generation         int32  `parquet:"name=generation, type=INT32"`
}

const (
	Hp             = "hp"
	Attack         = "attack"
	Defense        = "defense"
	SpecialAttack  = "special-attack"
	SpecialDefense = "special-defense"
	Speed          = "speed"
)

var expectedStats = []string{Hp, Attack, Defense, SpecialAttack, SpecialDefense, Speed}

func FromResponse(resp pokeapi.PokemonResponse) ([]Pokemon, error) {
	var result []Pokemon
	stats := make(map[string]int32)
	for _, stat := range resp.Stats {
		stats[stat.Stat.Name] = stat.BaseStat
	}
	var errs []error
	for _, stat := range expectedStats {
		_, ok := stats[stat]
		if !ok {
			errs = append(errs, errors.New(fmt.Sprintf("Stat %s not found", stat)))
		}
	}
	err := errors.Join(errs...)
	if err != nil {
		return nil, err
	}
	for _, pokeType := range resp.Types {
		entry := Pokemon{
			PokemonId:          resp.Id,
			Name:               resp.Name,
			Weight:             resp.Weight,
			Height:             resp.Height,
			Type:               pokeType.Type.Name,
			BaseHp:             stats[Hp],
			BaseAttack:         stats[Attack],
			BaseDefense:        stats[Defense],
			BaseSpecialAttack:  stats[SpecialAttack],
			BaseSpecialDefense: stats[SpecialDefense],
			BaseSpeed:          stats[Speed],
		}
		result = append(result, entry)
	}
	return result, nil
}
