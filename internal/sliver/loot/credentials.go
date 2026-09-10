package loot

import (
	"context"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"siren/internal/sliver/rpcwrap"
)

func (s *Service) GetCredentials() (*clientpb.Credentials, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.Creds, &commonpb.Empty{})
}

func (s *Service) AddCredential(username, plaintext, hash, collection string) error {
	_, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.CredsAdd, &clientpb.Credentials{
		Credentials: []*clientpb.Credential{{
			Username: username, Plaintext: plaintext,
			Hash: hash, Collection: collection,
		}},
	})
	if err != nil {
		return err
	}
	s.Publish("gui.loot-added", map[string]any{"kind": "credential", "username": username, "collection": collection})
	return nil
}

func (s *Service) RemoveCredential(id string) error {
	_, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.CredsRm, &clientpb.Credentials{
		Credentials: []*clientpb.Credential{{ID: id}},
	})
	return err
}

// UpdateCredentialRequest carries the mutable fields the credentials panel
// exposes for inline edits.
type UpdateCredentialRequest struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	Plaintext  string `json:"plaintext"`
	Hash       string `json:"hash"`
	HashType   int32  `json:"hashType"`
	Collection string `json:"collection"`
	IsCracked  bool   `json:"isCracked"`
}

// UpdateCredential rewrites a single credential row in-place. The server
// merges by ID, so any zero fields overwrite prior values — the UI must
// always send the full record it's showing.
func (s *Service) UpdateCredential(req UpdateCredentialRequest) error {
	c, err := rpcwrap.Client(s.rpc)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.ID) == "" {
		return fmt.Errorf("credential ID is required")
	}
	_, err = c.RPC().CredsUpdate(context.Background(), &clientpb.Credentials{
		Credentials: []*clientpb.Credential{{
			ID:         req.ID,
			Username:   req.Username,
			Plaintext:  req.Plaintext,
			Hash:       req.Hash,
			HashType:   clientpb.HashType(req.HashType),
			Collection: req.Collection,
			IsCracked:  req.IsCracked,
		}},
	})
	return err
}

// GetCredentialByID pulls a single credential — used by the detail drawer
// so we don't have to re-fetch the whole list to refresh one row.
func (s *Service) GetCredentialByID(id string) (*clientpb.Credential, error) {
	return rpcwrap.CallRequired(s.rpc, rpcpb.SliverRPCClient.GetCredByID, &clientpb.Credential{ID: id}, id, "credential ID")
}

// GetCredentialsByHashType filters server-side by hash algorithm — cheaper
// than pulling everything and filtering in the GUI when we've got 10k+ creds.
func (s *Service) GetCredentialsByHashType(hashType int32) (*clientpb.Credentials, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.GetCredsByHashType, &clientpb.Credential{
		HashType: clientpb.HashType(hashType),
	})
}

// GetPlaintextCredentialsByHashType returns only cracked (plaintext-known)
// creds of a given hash type — useful for spraying / pass-the-hash prep.
func (s *Service) GetPlaintextCredentialsByHashType(hashType int32) (*clientpb.Credentials, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.GetPlaintextCredsByHashType, &clientpb.Credential{
		HashType: clientpb.HashType(hashType),
	})
}

// SniffCredentialHashType asks the server to classify a raw hash blob —
// paste-and-detect flow so operators don't have to know the algorithm.
func (s *Service) SniffCredentialHashType(hash string) (*clientpb.Credential, error) {
	return rpcwrap.CallRequired(s.rpc, rpcpb.SliverRPCClient.CredsSniffHashType, &clientpb.Credential{Hash: hash}, hash, "hash")
}
