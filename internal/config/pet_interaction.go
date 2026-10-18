package config

import (
	"time"
	"whaleisle/internal/petcare"
)

func (s *Store) PokePet(now time.Time) error  { return s.applyPetCare("poke", petcare.Poke, now) }
func (s *Store) ThrowPet(now time.Time) error { return s.applyPetCare("throw", petcare.Throw, now) }
