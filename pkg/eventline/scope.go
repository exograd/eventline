package eventline

import (
	"fmt"

	"go.n16f.net/service/pkg/pg"
	"go.n16f.net/uuid"
)

type Scope interface {
	SQLCondition() string
	SQLCondition2(string) string
}

type GlobalScope struct {
}

func NewGlobalScope() *GlobalScope {
	return &GlobalScope{}
}

func (scope *GlobalScope) SQLCondition() string {
	return "TRUE"
}

func (scope *GlobalScope) SQLCondition2(correlation string) string {
	return "TRUE"
}

type AccountScope struct {
	AccountId uuid.UUID
}

func NewAccountScope(accountId uuid.UUID) Scope {
	return &AccountScope{
		AccountId: accountId,
	}
}

func (scope *AccountScope) SQLCondition() string {
	return "account_id=" + pg.QuoteString(scope.AccountId.String())
}

func (scope *AccountScope) SQLCondition2(correlation string) string {
	return correlation + ".account_id=" + pg.QuoteString(scope.AccountId.String())
}

type ProjectScope struct {
	ProjectId uuid.UUID
}

func NewProjectScope(projectId uuid.UUID) Scope {
	return &ProjectScope{
		ProjectId: projectId,
	}
}

func (scope *ProjectScope) SQLCondition() string {
	return "project_id=" + pg.QuoteString(scope.ProjectId.String())
}

func (scope *ProjectScope) SQLCondition2(correlation string) string {
	return correlation + ".project_id=" + pg.QuoteString(scope.ProjectId.String())
}

type AccountProjectScope struct {
	AccountId uuid.UUID
	ProjectId uuid.UUID
}

func NewAccountProjectScope(accountId, projectId uuid.UUID) Scope {
	return &AccountProjectScope{
		AccountId: accountId,
		ProjectId: projectId,
	}
}

func (scope *AccountProjectScope) SQLCondition() string {
	aid := pg.QuoteString(scope.AccountId.String())
	pid := pg.QuoteString(scope.ProjectId.String())

	return fmt.Sprintf("account_id=%s AND project_id=%s",
		aid, pid)
}

func (scope *AccountProjectScope) SQLCondition2(correlation string) string {
	aid := pg.QuoteString(scope.AccountId.String())
	pid := pg.QuoteString(scope.ProjectId.String())

	return fmt.Sprintf("%s.account_id=%s AND %s.project_id=%s",
		correlation, aid, correlation, pid)
}

type NullProjectScope struct {
}

func NewNullProjectScope() Scope {
	return &NullProjectScope{}
}

func (scope *NullProjectScope) SQLCondition() string {
	return "project_id IS NULL"
}

func (scope *NullProjectScope) SQLCondition2(correlation string) string {
	return correlation + ".project_id IS NULL"
}
