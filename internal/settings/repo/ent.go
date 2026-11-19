package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/secretsetting"
)

// Ent persists encrypted SecretSetting records via Ent.
type Ent struct {
	client *ent.Client
}

// New returns a new Ent repository for SecretSetting.
func New(client *ent.Client) *Ent {
	return &Ent{client: client}
}

// GetByKey returns the SecretSetting for key, or (nil, nil) if not found.
func (r *Ent) GetByKey(ctx context.Context, key string) (*ent.SecretSetting, error) {
	rec, err := r.client.SecretSetting.Query().
		Where(secretsetting.KeyEQ(key)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("query secret setting %q: %w", key, err)
	}
	return rec, nil
}

// Upsert creates or updates the encrypted SecretSetting for key.
func (r *Ent) Upsert(
	ctx context.Context,
	key string,
	ciphertext, nonce []byte,
	keyVersion int,
	last4 string,
) (*ent.SecretSetting, error) {
	existing, err := r.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		rec, err := r.client.SecretSetting.Create().
			SetKey(key).
			SetCiphertext(ciphertext).
			SetNonce(nonce).
			SetKeyVersion(keyVersion).
			SetLast4(last4).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("create secret setting %q: %w", key, err)
		}
		return rec, nil
	}

	rec, err := existing.Update().
		SetCiphertext(ciphertext).
		SetNonce(nonce).
		SetKeyVersion(keyVersion).
		SetLast4(last4).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update secret setting %q: %w", key, err)
	}
	return rec, nil
}

// DeleteByKey removes the SecretSetting for key if it exists.
func (r *Ent) DeleteByKey(ctx context.Context, key string) error {
	_, err := r.client.SecretSetting.Delete().
		Where(secretsetting.KeyEQ(key)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete secret setting %q: %w", key, err)
	}
	return nil
}
