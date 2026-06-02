package api

import (
	z "github.com/Oudwins/zog"
	"go.lumeweb.com/httputil"
)

var (
	_ httputil.DTOValidator                    = (*ProfilingStatusResponse)(nil)
	_ httputil.DTOResponse[*ProfilingStatusResponse] = (*ProfilingStatusResponse)(nil)

	_ httputil.DTOValidator               = (*BlockProfileRequest)(nil)
	_ httputil.DTORequest[*BlockProfileRequest] = (*BlockProfileRequest)(nil)

	_ httputil.DTOValidator               = (*MutexProfileRequest)(nil)
	_ httputil.DTORequest[*MutexProfileRequest] = (*MutexProfileRequest)(nil)
)

type ProfilingStatusResponse struct {
	BlockProfileRate int `json:"block_profile_rate"`
	MutexFraction   int `json:"mutex_fraction"`
}

func (r *ProfilingStatusResponse) Schema() *z.StructSchema {
	return z.Struct(z.Shape{
		"BlockProfileRate": z.Int().Required(),
		"MutexFraction":   z.Int().Required(),
	})
}

func (r *ProfilingStatusResponse) FromModel(model *ProfilingStatusResponse) error {
	r.BlockProfileRate = model.BlockProfileRate
	r.MutexFraction = model.MutexFraction
	return nil
}

type BlockProfileRequest struct {
	Rate int `json:"rate"`
}

func (r *BlockProfileRequest) Schema() *z.StructSchema {
	return z.Struct(z.Shape{
		"Rate": z.Int().Required().GTE(0),
	})
}

func (r *BlockProfileRequest) ToModel() (*BlockProfileRequest, error) {
	return r, nil
}

type MutexProfileRequest struct {
	Fraction int `json:"fraction"`
}

func (r *MutexProfileRequest) Schema() *z.StructSchema {
	return z.Struct(z.Shape{
		"Fraction": z.Int().Required().GTE(0),
	})
}

func (r *MutexProfileRequest) ToModel() (*MutexProfileRequest, error) {
	return r, nil
}
