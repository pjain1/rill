package payment

import (
	"context"

	"github.com/rilldata/rill/admin/database"
)

var _ Provider = &noop{}

type noop struct{}

func NewNoop() Provider {
	return noop{}
}

func (n noop) CreateCustomer(ctx context.Context, organization *database.Organization) (*Customer, error) {
	return &Customer{}, nil
}

func (n noop) FindCustomer(ctx context.Context, customerID string) (*Customer, error) {
	return &Customer{}, nil
}

func (n noop) FindCustomerForOrg(ctx context.Context, organization *database.Organization) (*Customer, error) {
	return &Customer{}, nil
}

func (n noop) DeleteCustomer(ctx context.Context, customerID string) error {
	return nil
}

func (n noop) GetBillingPortalURL(ctx context.Context, customerID, returnURL string) (string, error) {
	return "", nil
}
