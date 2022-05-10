package db

import (
	"database/sql"
	"fmt"

	_ "embed"

	"github.com/jmoiron/sqlx"
)

//go:embed seeds/appuio_cloud_memory.promql
var appuioCloudMemoryQuery string

//go:embed seeds/appuio_cloud_memory_sub_memory.promql
var appuioCloudMemorySubQueryMemory string

//go:embed seeds/appuio_cloud_memory_sub_cpu.promql
var appuioCloudMemorySubQueryCPU string

//go:embed seeds/appuio_cloud_loadbalancer.promql
var appuioCloudLoadbalancerQuery string

//go:embed seeds/appuio_cloud_persistent_storage.promql
var appuioCloudPersistentStorageQuery string

// DefaultQueries consists of default starter queries.
var DefaultQueries = []Query{
	{
		Name:        "appuio_cloud_memory",
		Description: "Memory usage (maximum of requested and used memory) aggregated by namespace",
		DisplayName: "Compute",
		Query:       appuioCloudMemoryQuery,
		Unit:        "MiB",
		subQueries: []Query{
			{
				Name:        "appuio_cloud_memory_subquery_memory_request",
				Description: "Memory request aggregated by namespace",
				DisplayName: "Memory Requests",
				Query:       appuioCloudMemorySubQueryMemory,
				Unit:        "MiB",
			},
			{
				Name:        "appuio_cloud_memory_subquery_cpu_request",
				Description: "CPU requests exceeding the fair use limit, converted to the memory request equivalent",
				DisplayName: "CPU Requests",
				Query:       appuioCloudMemorySubQueryCPU,
				Unit:        "MiB",
			},
		},
	},
	{
		Name:        "appuio_cloud_loadbalancer",
		Description: "Number of services of type load balancer",
		DisplayName: "Service Type LoadBalancer",
		Query:       appuioCloudLoadbalancerQuery,
	},
	{
		Name:        "appuio_cloud_persistent_storage",
		Description: "Persistent storage usage aggregated by namespace and storageclass",
		DisplayName: "Persistent Storage",
		Query:       appuioCloudPersistentStorageQuery,
		Unit:        "GiB",
	},
}

// Seed seeds the database with "starter" data.
// Is idempotent and thus can be executed multiple times in one database.
func Seed(db *sql.DB) error {
	return SeedQueries(db, DefaultQueries)
}

func SeedQueries(db *sql.DB, queries []Query) error {
	dbx := NewDBx(db)
	tx, err := dbx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := createQueries(tx, queries); err != nil {
		return err
	}

	return tx.Commit()
}

func createQueries(tx *sqlx.Tx, queries []Query) error {
	for _, q := range queries {
		exists, err := queryExistsByName(tx, q.Name)
		if err != nil {
			return fmt.Errorf("error checking if query exists: %w", err)
		}
		if exists {
			fmt.Printf("Found query with name '%s'. Skip creating default query.\n", q.Name)
			continue
		}

		err = GetNamed(tx, &q.Id,
			"INSERT INTO queries (name,description,query,unit,during) VALUES (:name,:description,:query,:unit,'[-infinity,infinity)') RETURNING id",
			q)
		if err != nil {
			return fmt.Errorf("error creating default query: %w", err)
		}

		for _, subQuery := range q.subQueries {
			subQuery.ParentID = sql.NullString{
				String: q.Id,
				Valid:  true,
			}
			exists, err := queryExistsByName(tx, subQuery.Name)
			if err != nil {
				return fmt.Errorf("error checking if sub-query exists: %w", err)
			}
			if exists {
				fmt.Printf("Found sub-query with name '%s'. Skip creating default query.\n", subQuery.Name)
				continue
			}
			_, err = tx.NamedExec("INSERT INTO queries (name,description,query,unit,during) VALUES (:name,:description,:query,:unit,'[-infinity,infinity)')", subQuery)
			if err != nil {
				return fmt.Errorf("error creating default sub-query: %w", err)
			}
		}
	}
	return nil
}

func queryExistsByName(tx *sqlx.Tx, name string) (bool, error) {
	var exists bool
	err := tx.Get(&exists, "SELECT EXISTS(SELECT 1 FROM queries WHERE name = $1)", name)
	return exists, err
}
