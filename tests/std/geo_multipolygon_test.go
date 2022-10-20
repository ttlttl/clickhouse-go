// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package std

import (
	"context"
	"fmt"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/paulmach/orb"
	"github.com/stretchr/testify/assert"
)

func TestStdGeoMultiPolygon(t *testing.T) {
	ctx := clickhouse.Context(context.Background(), clickhouse.WithSettings(clickhouse.Settings{
		"allow_experimental_geo_types": 1,
	}))

	dsns := map[string]clickhouse.Protocol{"Native": clickhouse.Native, "Http": clickhouse.HTTP}
	useSSL, err := strconv.ParseBool(clickhouse_tests.GetEnv("CLICKHOUSE_USE_SSL", "false"))
	require.NoError(t, err)
	for name, protocol := range dsns {
		t.Run(fmt.Sprintf("%s Protocol", name), func(t *testing.T) {
			conn, err := GetStdDSNConnection(protocol, useSSL, "false")
			require.NoError(t, err)
			if !CheckMinServerVersion(conn, 21, 12, 0) {
				t.Skip(fmt.Errorf("unsupported clickhouse version"))
				return
			}
			const ddl = `
				CREATE TABLE test_geo_multipolygon (
					  Col1 MultiPolygon
					, Col2 Array(MultiPolygon)
				) Engine MergeTree() ORDER BY tuple()
				`
			defer func() {
				conn.Exec("DROP TABLE test_geo_multipolygon")
			}()
			_, err = conn.ExecContext(ctx, ddl)
			require.NoError(t, err)
			scope, err := conn.Begin()
			require.NoError(t, err)
			batch, err := scope.Prepare("INSERT INTO test_geo_multipolygon")
			require.NoError(t, err)
			var (
				col1Data = orb.MultiPolygon{
					orb.Polygon{
						orb.Ring{
							orb.Point{1, 2},
							orb.Point{12, 2},
						},
						orb.Ring{
							orb.Point{11, 2},
							orb.Point{1, 12},
						},
					},
					orb.Polygon{
						orb.Ring{
							orb.Point{1, 2},
							orb.Point{12, 2},
						},
						orb.Ring{
							orb.Point{11, 2},
							orb.Point{1, 12},
						},
					},
				}
				col2Data = []orb.MultiPolygon{
					[]orb.Polygon{
						[]orb.Ring{
							orb.Ring{
								orb.Point{1, 2},
								orb.Point{1, 22},
							},
							orb.Ring{
								orb.Point{1, 23},
								orb.Point{12, 2},
							},
						},
						[]orb.Ring{
							orb.Ring{
								orb.Point{21, 2},
								orb.Point{1, 222},
							},
							orb.Ring{
								orb.Point{21, 23},
								orb.Point{12, 22},
							},
						},
					},
					[]orb.Polygon{
						[]orb.Ring{
							orb.Ring{
								orb.Point{11, 2},
								orb.Point{1, 22},
							},
							orb.Ring{
								orb.Point{1, 23},
								orb.Point{12, 22},
							},
						},
						[]orb.Ring{
							orb.Ring{
								orb.Point{21, 2},
								orb.Point{1, 222},
							},
							orb.Ring{
								orb.Point{21, 23},
								orb.Point{12, 22},
							},
						},
					},
				}
			)
			_, err = batch.Exec(col1Data, col2Data)
			require.NoError(t, err)
			require.NoError(t, scope.Commit())
			var (
				col1 orb.MultiPolygon
				col2 []orb.MultiPolygon
			)
			require.NoError(t, conn.QueryRow("SELECT * FROM test_geo_multipolygon").Scan(&col1, &col2))
			assert.Equal(t, col1Data, col1)
			assert.Equal(t, col2Data, col2)
		})
	}
}
