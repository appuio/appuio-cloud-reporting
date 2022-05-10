package invoice_test

import (
	"database/sql"
	"time"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"

	"github.com/stretchr/testify/require"
)

func (s *InvoiceGoldenSuite) TestInvoiceGolden_TimedQuery() {
	t := s.T()
	tdb := s.DB()

	// Create base product and discount
	_, err := db.CreateProduct(tdb, db.Product{
		Source: "my-product",
		Amount: 1,
		During: db.InfiniteRange(),
	})
	require.NoError(t, err)
	_, err = db.CreateDiscount(tdb, db.Discount{
		Source: "my-product",
		During: db.InfiniteRange(),
	})
	require.NoError(t, err)

	// Create old query, only valid before the billing period.
	// Should not be in invoice
	old, err := db.CreateQuery(tdb, db.Query{
		Name:        "test",
		Description: "old invalid query",
		DisplayName: "Old Query",
		Query:       "old-test",
		Unit:        "tps",
		During:      timerange(t, "-", "2022-02-25"),
	})
	s.prom.queries[old.Query] = fakeQueryResults{
		"my-product:my-cluster:my-tenant:my-namespace":    fakeQuerySample{Value: 9001},
		"my-product:my-cluster:other-tenant:my-namespace": fakeQuerySample{Value: 9001},
	}
	require.NoError(t, err)

	// Create query and two subqueries that are valid for the first 5 days
	// One subquery is only valid for the first two days of the billing month
	q, err := db.CreateQuery(tdb, db.Query{
		Name:        "test",
		Description: "test description",
		DisplayName: "New Query",
		Query:       "test",
		Unit:        "tps",
		During:      timerange(t, "2022-02-25", "2022-03-05"),
	})
	s.prom.queries[q.Query] = fakeQueryResults{
		"my-product:my-cluster:my-tenant:my-namespace":    fakeQuerySample{Value: 42},
		"my-product:my-cluster:other-tenant:my-namespace": fakeQuerySample{Value: 23},
	}
	require.NoError(t, err)
	sq, err := db.CreateQuery(tdb, db.Query{
		ParentID: sql.NullString{
			String: q.Id,
			Valid:  true,
		},
		Name:        "sub-test",
		Description: "A sub query of Test",
		DisplayName: "New Sub Query",
		Query:       "sub-test",
		Unit:        "tps",
		During:      db.InfiniteRange(),
	})
	s.prom.queries[sq.Query] = fakeQueryResults{
		"my-product:my-cluster:my-tenant:my-namespace":    fakeQuerySample{Value: 4},
		"my-product:my-cluster:other-tenant:my-namespace": fakeQuerySample{Value: 2},
	}
	require.NoError(t, err)
	sq2, err := db.CreateQuery(tdb, db.Query{
		ParentID: sql.NullString{
			String: q.Id,
			Valid:  true,
		},
		Name:        "sub-test2",
		Description: "An other sub query of Test that stops early",
		DisplayName: "Other new Sub Query",
		Query:       "sub-test2",
		Unit:        "tps",
		During:      timerange(t, "2022-02-25", "2022-03-02"),
	})
	s.prom.queries[sq2.Query] = fakeQueryResults{
		"my-product:my-cluster:my-tenant:my-namespace":    fakeQuerySample{Value: 7},
		"my-product:my-cluster:other-tenant:my-namespace": fakeQuerySample{Value: 0},
	}
	require.NoError(t, err)

	// Create new query that is valid from the 5th day and has one subquery
	newQ, err := db.CreateQuery(tdb, db.Query{
		Name:        "test",
		Description: "new nicer query",
		DisplayName: "Better Query",
		Query:       "nice-test",
		Unit:        "tps",
		During:      timerange(t, "2022-03-05", "-"),
	})
	s.prom.queries[newQ.Query] = fakeQueryResults{
		"my-product:my-cluster:my-tenant:my-namespace":    fakeQuerySample{Value: 69},
		"my-product:my-cluster:other-tenant:my-namespace": fakeQuerySample{Value: 69},
	}
	require.NoError(t, err)
	nsq, err := db.CreateQuery(tdb, db.Query{
		ParentID: sql.NullString{
			String: newQ.Id,
			Valid:  true,
		},
		Name:        "new-sub-test",
		Description: "A better sub query of Test",
		DisplayName: "Simpler Sub Query",
		Query:       "new-sub-test",
		Unit:        "tps",
		During:      db.InfiniteRange(),
	})
	s.prom.queries[nsq.Query] = fakeQueryResults{
		"my-product:my-cluster:my-tenant:my-namespace":    fakeQuerySample{Value: 4},
		"my-product:my-cluster:other-tenant:my-namespace": fakeQuerySample{Value: 2},
	}
	require.NoError(t, err)

	runReport(t, tdb, s.prom, "test", "2022-02-25", "2022-03-10")
	invoiceEqualsGolden(t, "timed_query",
		generateInvoice(t, tdb, 2022, time.March),
		*updateGolden)
}
