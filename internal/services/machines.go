package services

import (
	"github.com/kern/internal/models"
	nundb "github.com/viewfromaside/nun-db-go"
)

type MachineService struct {
	machines []*models.Machine
	nundb    *nundb.Client
}

func NewMachineService(nundb *nundb.Client) *MachineService {
	return &MachineService{
		nundb: nundb,
	}
}

func (ms *MachineService) Create(machine models.Machine) error {
	ms.nundb.Set(machine.ID, machine)
	ms.machines = append(ms.machines, &machine)
	return nil
}

func (ms *MachineService) GetAll() ([]*models.Machine, error) {
	// result, err := ms.nundb.Get("")
	// if err != nil {
	// 	return nil, err
	// }
	return ms.machines, nil
}
