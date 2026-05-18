package entgo

import (
	"context"
	"fmt"

	"github.com/casbin/casbin/v2/model"
	"github.com/kilip/opus/server/ent"
	"github.com/kilip/opus/server/ent/casbinrule"
)

// CasbinAdapter maps standard Ent database clients to satisfy persist.Adapter.
type CasbinAdapter struct {
	client *ent.Client
}

// NewCasbinAdapter creates a database policy connector.
func NewCasbinAdapter(client *ent.Client) *CasbinAdapter {
	return &CasbinAdapter{client: client}
}

// loadPolicyLine loads a single policy rule line into Casbin model.
func loadPolicyLine(line *ent.CasbinRule, model model.Model) {
	key := line.Ptype
	sec := key[:1]

	tokens := []string{}
	if line.V0 != "" {
		tokens = append(tokens, line.V0)
	}
	if line.V1 != "" {
		tokens = append(tokens, line.V1)
	}
	if line.V2 != "" {
		tokens = append(tokens, line.V2)
	}
	if line.V3 != "" {
		tokens = append(tokens, line.V3)
	}
	if line.V4 != "" {
		tokens = append(tokens, line.V4)
	}
	if line.V5 != "" {
		tokens = append(tokens, line.V5)
	}

	if _, ok := model[sec]; ok {
		if _, ok := model[sec][key]; ok {
			model[sec][key].Policy = append(model[sec][key].Policy, tokens)
		}
	}
}

// LoadPolicy retrieves policy lines from active database connections.
func (a *CasbinAdapter) LoadPolicy(model model.Model) error {
	ctx := context.Background()
	lines, err := a.client.CasbinRule.Query().All(ctx)
	if err != nil {
		return fmt.Errorf("casbin.adapter.LoadPolicy: %w", err)
	}

	for _, line := range lines {
		loadPolicyLine(line, model)
	}
	return nil
}

// SavePolicy commits loaded Casbin model changes (truncates and bulk-recreates rules).
func (a *CasbinAdapter) SavePolicy(model model.Model) error {
	ctx := context.Background()
	tx, err := a.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("casbin.adapter.SavePolicy: %w", err)
	}

	// Delete all existing rules
	_, err = tx.CasbinRule.Delete().Exec(ctx)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("casbin.adapter.SavePolicy: failed to clear existing rules: %w", err)
	}

	// Insert new rules
	var bulk []*ent.CasbinRuleCreate
	for ptype, ast := range model["p"] {
		for _, rule := range ast.Policy {
			line := a.savePolicyLine(tx, ptype, rule)
			bulk = append(bulk, line)
		}
	}
	for ptype, ast := range model["g"] {
		for _, rule := range ast.Policy {
			line := a.savePolicyLine(tx, ptype, rule)
			bulk = append(bulk, line)
		}
	}

	if len(bulk) > 0 {
		err = tx.CasbinRule.CreateBulk(bulk...).Exec(ctx)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("casbin.adapter.SavePolicy: failed to bulk save rules: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("casbin.adapter.SavePolicy: %w", err)
	}
	return nil
}

func (a *CasbinAdapter) savePolicyLine(tx *ent.Tx, ptype string, rule []string) *ent.CasbinRuleCreate {
	line := tx.CasbinRule.Create().SetPtype(ptype)
	if len(rule) > 0 {
		line.SetV0(rule[0])
	}
	if len(rule) > 1 {
		line.SetV1(rule[1])
	}
	if len(rule) > 2 {
		line.SetV2(rule[2])
	}
	if len(rule) > 3 {
		line.SetV3(rule[3])
	}
	if len(rule) > 4 {
		line.SetV4(rule[4])
	}
	if len(rule) > 5 {
		line.SetV5(rule[5])
	}
	return line
}

// AddPolicy appends a policy rule.
func (a *CasbinAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	ctx := context.Background()
	create := a.client.CasbinRule.Create().SetPtype(ptype)
	if len(rule) > 0 {
		create.SetV0(rule[0])
	}
	if len(rule) > 1 {
		create.SetV1(rule[1])
	}
	if len(rule) > 2 {
		create.SetV2(rule[2])
	}
	if len(rule) > 3 {
		create.SetV3(rule[3])
	}
	if len(rule) > 4 {
		create.SetV4(rule[4])
	}
	if len(rule) > 5 {
		create.SetV5(rule[5])
	}

	_, err := create.Save(ctx)
	if err != nil {
		return fmt.Errorf("casbin.adapter.AddPolicy: %w", err)
	}
	return nil
}

// RemovePolicy deletes target rules.
func (a *CasbinAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	ctx := context.Background()
	query := a.client.CasbinRule.Delete().Where(casbinrule.PtypeEQ(ptype))
	if len(rule) > 0 {
		query = query.Where(casbinrule.V0EQ(rule[0]))
	}
	if len(rule) > 1 {
		query = query.Where(casbinrule.V1EQ(rule[1]))
	}
	if len(rule) > 2 {
		query = query.Where(casbinrule.V2EQ(rule[2]))
	}
	if len(rule) > 3 {
		query = query.Where(casbinrule.V3EQ(rule[3]))
	}
	if len(rule) > 4 {
		query = query.Where(casbinrule.V4EQ(rule[4]))
	}
	if len(rule) > 5 {
		query = query.Where(casbinrule.V5EQ(rule[5]))
	}

	_, err := query.Exec(ctx)
	if err != nil {
		return fmt.Errorf("casbin.adapter.RemovePolicy: %w", err)
	}
	return nil
}

// RemoveFilteredPolicy deletes filtered elements.
func (a *CasbinAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	ctx := context.Background()
	query := a.client.CasbinRule.Delete().Where(casbinrule.PtypeEQ(ptype))

	if fieldIndex <= 0 && 0 < fieldIndex+len(fieldValues) && fieldValues[0-fieldIndex] != "" {
		query = query.Where(casbinrule.V0EQ(fieldValues[0-fieldIndex]))
	}
	if fieldIndex <= 1 && 1 < fieldIndex+len(fieldValues) && fieldValues[1-fieldIndex] != "" {
		query = query.Where(casbinrule.V1EQ(fieldValues[1-fieldIndex]))
	}
	if fieldIndex <= 2 && 2 < fieldIndex+len(fieldValues) && fieldValues[2-fieldIndex] != "" {
		query = query.Where(casbinrule.V2EQ(fieldValues[2-fieldIndex]))
	}
	if fieldIndex <= 3 && 3 < fieldIndex+len(fieldValues) && fieldValues[3-fieldIndex] != "" {
		query = query.Where(casbinrule.V3EQ(fieldValues[3-fieldIndex]))
	}
	if fieldIndex <= 4 && 4 < fieldIndex+len(fieldValues) && fieldValues[4-fieldIndex] != "" {
		query = query.Where(casbinrule.V4EQ(fieldValues[4-fieldIndex]))
	}
	if fieldIndex <= 5 && 5 < fieldIndex+len(fieldValues) && fieldValues[5-fieldIndex] != "" {
		query = query.Where(casbinrule.V5EQ(fieldValues[5-fieldIndex]))
	}

	_, err := query.Exec(ctx)
	if err != nil {
		return fmt.Errorf("casbin.adapter.RemoveFilteredPolicy: %w", err)
	}
	return nil
}
