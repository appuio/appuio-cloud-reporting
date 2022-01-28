package invoice_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/appuio/appuio-cloud-reporting/pkg/db/dbtest"
	"github.com/appuio/appuio-cloud-reporting/pkg/invoice"
)

type InvoiceSuite struct {
	dbtest.Suite

	memoryProduct  db.Product
	storageProduct db.Product

	memoryDiscount  db.Discount
	storageDiscount db.Discount

	memoryQuery  db.Query
	storageQuery db.Query

	umbrellaCorpTenant db.Tenant
	tricellTenant      db.Tenant

	p12aCategory         db.Category
	nestElevCtrlCategory db.Category
	uroborosCategory     db.Category

	dateTimes []db.DateTime

	facts []db.Fact
}

func (s *InvoiceSuite) SetupSuite() {
	s.Suite.SetupSuite()

	t := s.T()
	tdb := s.DB()

	require.NoError(s.T(),
		db.GetNamed(tdb, &s.memoryProduct,
			"INSERT INTO products (source,target,amount,unit,during) VALUES (:source,:target,:amount,:unit,:during) RETURNING *", db.Product{
				Source: "test_memory:us-rac-2",
				Amount: 3,
				During: db.InfiniteRange(),
			}))
	require.NoError(s.T(),
		db.GetNamed(tdb, &s.storageProduct,
			"INSERT INTO products (source,target,amount,unit,during) VALUES (:source,:target,:amount,:unit,:during) RETURNING *", db.Product{
				Source: "test_storage:us-rac-2",
				Amount: 5,
				During: db.InfiniteRange(),
			}))

	require.NoError(t,
		db.GetNamed(tdb, &s.memoryDiscount,
			"INSERT INTO discounts (source,discount,during) VALUES (:source,:discount,:during) RETURNING *", db.Discount{
				Source:   "test_memory:us-rac-2",
				Discount: 0,
				During:   db.InfiniteRange(),
			}))
	require.NoError(t,
		db.GetNamed(tdb, &s.storageDiscount,
			"INSERT INTO discounts (source,discount,during) VALUES (:source,:discount,:during) RETURNING *", db.Discount{
				Source:   "test_storage:us-rac-2",
				Discount: 0.4,
				During:   db.InfiniteRange(),
			}))

	require.NoError(t,
		db.GetNamed(tdb, &s.memoryQuery,
			"INSERT INTO queries (name,description,unit,query) VALUES (:name,:description,:unit,:query) RETURNING *", db.Query{
				Name:        "test_memory",
				Description: "Memory",
				Unit:        "MiB",
			}))
	require.NoError(t,
		db.GetNamed(tdb, &s.storageQuery,
			"INSERT INTO queries (name,description,unit,query) VALUES (:name,:description,:unit,:query) RETURNING *", db.Query{
				Name:        "test_storage",
				Description: "Storage",
				Unit:        "Gib",
			}))

	require.NoError(t,
		db.GetNamed(tdb, &s.umbrellaCorpTenant,
			"INSERT INTO tenants (source,target) VALUES (:source,:target) RETURNING *", db.Tenant{
				Source: "umbrellacorp",
				Target: sql.NullString{Valid: true, String: "23465-umbrellacorp"},
			}))
	require.NoError(t,
		db.GetNamed(tdb, &s.tricellTenant,
			"INSERT INTO tenants (source,target) VALUES (:source,:target) RETURNING *", db.Tenant{
				Source: "tricell",
				Target: sql.NullString{Valid: true, String: "98756-tricell"},
			}))

	require.NoError(t,
		db.GetNamed(tdb, &s.p12aCategory,
			"INSERT INTO categories (source,target) VALUES (:source,:target) RETURNING *", db.Category{
				Source: "us-rac-2:disposal-plant-p-12a-furnace-control",
				Target: sql.NullString{Valid: true, String: "3445-disposal-plant-p-12a-furnace-control"},
			}))
	require.NoError(t,
		db.GetNamed(tdb, &s.nestElevCtrlCategory,
			"INSERT INTO categories (source,target) VALUES (:source,:target) RETURNING *", db.Category{
				Source: "us-rac-2:nest-elevator-control",
				Target: sql.NullString{Valid: true, String: "897-nest-elevator-control"},
			}))
	require.NoError(t,
		db.GetNamed(tdb, &s.uroborosCategory,
			"INSERT INTO categories (source,target) VALUES (:source,:target) RETURNING *", db.Category{
				Source: "af-south-1:uroboros-research",
				Target: sql.NullString{Valid: true, String: "123587-uroboros-research"},
			}))

	require.NoError(t,
		db.SelectNamed(tdb, &s.dateTimes,
			"INSERT INTO date_times (timestamp, year, month, day, hour) VALUES (:timestamp, :year, :month, :day, :hour) RETURNING *",
			[]db.DateTime{
				db.BuildDateTime(time.Date(2021, time.December, 1, 1, 0, 0, 0, time.UTC)),
				db.BuildDateTime(time.Date(2021, time.December, 31, 23, 0, 0, 0, time.UTC)),
				db.BuildDateTime(time.Date(2022, time.January, 1, 1, 0, 0, 0, time.UTC)),
			},
		))

	facts := make([]db.Fact, 0)

	facts = append(facts, factWithDateTime(db.Fact{
		QueryId:    s.memoryQuery.Id,
		ProductId:  s.memoryProduct.Id,
		DiscountId: s.memoryDiscount.Id,

		TenantId:   s.umbrellaCorpTenant.Id,
		CategoryId: s.p12aCategory.Id,

		Quantity: 4000,
	}, s.dateTimes)...)

	facts = append(facts, factWithDateTime(db.Fact{
		QueryId:    s.storageQuery.Id,
		ProductId:  s.storageProduct.Id,
		DiscountId: s.storageDiscount.Id,

		TenantId:   s.umbrellaCorpTenant.Id,
		CategoryId: s.p12aCategory.Id,

		Quantity: 12,
	}, s.dateTimes)...)

	facts = append(facts, factWithDateTime(db.Fact{
		QueryId:    s.memoryQuery.Id,
		ProductId:  s.memoryProduct.Id,
		DiscountId: s.memoryDiscount.Id,

		TenantId:   s.umbrellaCorpTenant.Id,
		CategoryId: s.nestElevCtrlCategory.Id,

		Quantity: 1000,
	}, s.dateTimes)...)

	facts = append(facts, factWithDateTime(db.Fact{
		QueryId:    s.memoryQuery.Id,
		ProductId:  s.memoryProduct.Id,
		DiscountId: s.memoryDiscount.Id,

		TenantId:   s.tricellTenant.Id,
		CategoryId: s.uroborosCategory.Id,

		Quantity: 2000,
	}, s.dateTimes)...)

	require.NoError(t,
		db.SelectNamed(tdb, &s.facts,
			"INSERT INTO facts (date_time_id,query_id,tenant_id,category_id,product_id,discount_id,quantity) VALUES (:date_time_id,:query_id,:tenant_id,:category_id,:product_id,:discount_id,:quantity) RETURNING *",
			facts,
		))
}

func (s *InvoiceSuite) TestInvoice_Generate() {
	t := s.T()

	tx, err := s.DB().Beginx()
	require.NoError(t, err)
	defer tx.Rollback()

	invRun, err := invoice.Generate(context.Background(), tx, 2021, time.December)
	require.NoError(t, err)
	require.Len(t, invRun.Invoices, 2)

	require.Equal(t, time.Date(2021, time.December, 1, 0, 0, 0, 0, time.UTC), invRun.PeriodStart)
	require.Equal(t, time.Date(2021, time.December, 31, 0, 0, 0, 0, time.UTC), invRun.PeriodEnd)

	discountToMultiplier := func(discount float64) float64 {
		return float64(1) - float64(discount)
	}

	const stampsInTimerange = 2
	t.Run("InvoiceForTricell", func(t *testing.T) {
		inv := invRun.Invoices[0]
		const quantity = float64(2000) * stampsInTimerange
		total := quantity * s.memoryProduct.Amount * discountToMultiplier(s.memoryDiscount.Discount)

		require.Equal(t, invoice.Invoice{
			Tenant: invoice.Tenant{
				ID:     s.tricellTenant.Id,
				Source: s.tricellTenant.Source,
				Target: s.tricellTenant.Target.String,
			},
			Categories: []invoice.Category{
				{
					ID:     s.uroborosCategory.Id,
					Source: s.uroborosCategory.Source,
					Target: s.uroborosCategory.Target.String,
					Items: []invoice.Item{
						{
							Description:  s.memoryQuery.Description,
							Quantity:     quantity,
							Unit:         s.memoryProduct.Unit,
							PricePerUnit: s.memoryProduct.Amount,
							Discount:     s.memoryDiscount.Discount,
							Total:        total,
						},
					},
					Total: total,
				},
			},
			Total: total,
		}, inv)
	})

	t.Run("InvoiceForUmbrellaCorp", func(t *testing.T) {
		inv := invRun.Invoices[1]
		const memP12Quantity = float64(4000) * stampsInTimerange
		memP12Total := memP12Quantity * s.memoryProduct.Amount * discountToMultiplier(s.memoryDiscount.Discount)
		const storP12Quantity = float64(12) * stampsInTimerange
		storP12Total := storP12Quantity * s.storageProduct.Amount * discountToMultiplier(s.storageDiscount.Discount)
		const memNestQuantity = float64(1000) * stampsInTimerange
		memNestTotal := memNestQuantity * s.memoryProduct.Amount * discountToMultiplier(s.memoryDiscount.Discount)

		require.Equal(t, invoice.Invoice{
			Tenant: invoice.Tenant{
				ID:     s.umbrellaCorpTenant.Id,
				Source: s.umbrellaCorpTenant.Source,
				Target: s.umbrellaCorpTenant.Target.String,
			},
			Categories: []invoice.Category{
				{
					ID:     s.p12aCategory.Id,
					Source: s.p12aCategory.Source,
					Target: s.p12aCategory.Target.String,
					Items: []invoice.Item{
						{
							Description:  s.memoryQuery.Description,
							Quantity:     memP12Quantity,
							Unit:         s.memoryProduct.Unit,
							PricePerUnit: s.memoryProduct.Amount,
							Discount:     s.memoryDiscount.Discount,
							Total:        memP12Total,
						},
						{
							Description:  s.storageQuery.Description,
							Quantity:     storP12Quantity,
							Unit:         s.storageProduct.Unit,
							PricePerUnit: s.storageProduct.Amount,
							Discount:     s.storageDiscount.Discount,
							Total:        storP12Total,
						},
					},
					Total: memP12Total + storP12Total,
				},
				{
					ID:     s.nestElevCtrlCategory.Id,
					Source: s.nestElevCtrlCategory.Source,
					Target: s.nestElevCtrlCategory.Target.String,
					Items: []invoice.Item{
						{
							Description:  s.memoryQuery.Description,
							Quantity:     memNestQuantity,
							Unit:         s.memoryProduct.Unit,
							PricePerUnit: s.memoryProduct.Amount,
							Discount:     s.memoryDiscount.Discount,
							Total:        memNestTotal,
						},
					},
					Total: memNestTotal,
				},
			},
			Total: memP12Total + storP12Total + memNestTotal,
		}, inv)
	})
}

func TestInvoice(t *testing.T) {
	suite.Run(t, new(InvoiceSuite))
}

func factWithDateTime(f db.Fact, dts []db.DateTime) []db.Fact {
	facts := make([]db.Fact, 0, len(dts))
	for _, dt := range dts {
		f.DateTimeId = dt.Id
		facts = append(facts, f)
	}
	return facts
}