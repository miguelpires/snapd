// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2025 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package confdb_test

import (
	"github.com/snapcore/snapd/confdb"
	. "gopkg.in/check.v1"
)

type filtersSuite struct{}

var _ = Suite(&filtersSuite{})

func (*filtersSuite) TestParsePathWithFilters(c *C) {
	type testcase struct {
		path            string
		expectedSubkeys []string
		expectedFilters map[int][]confdb.FieldFilter
		err             string
	}

	tcs := []testcase{
		{
			path:            "foo.bar[.a={b}]",
			expectedSubkeys: []string{"foo", "bar"},
			expectedFilters: map[int][]confdb.FieldFilter{
				1: {{Field: "a", Placeholder: "b"}},
			},
		},
		{
			path:            "foo[.x={y}].bar",
			expectedSubkeys: []string{"foo", "bar"},
			expectedFilters: map[int][]confdb.FieldFilter{
				0: {{Field: "x", Placeholder: "y"}},
			},
		},
		{
			path:            "foo[.a={b}][.c={d}].bar",
			expectedSubkeys: []string{"foo", "bar"},
			expectedFilters: map[int][]confdb.FieldFilter{
				0: {
					{Field: "a", Placeholder: "b"},
					{Field: "c", Placeholder: "d"},
				},
			},
		},
		{
			path:            "foo[.x={y}].bar[.a={b}]",
			expectedSubkeys: []string{"foo", "bar"},
			expectedFilters: map[int][]confdb.FieldFilter{
				0: {{Field: "x", Placeholder: "y"}},
				1: {{Field: "a", Placeholder: "b"}},
			},
		},
		{
			path:            "a.b.c",
			expectedSubkeys: []string{"a", "b", "c"},
			expectedFilters: map[int][]confdb.FieldFilter{},
		},
		{
			path:            "foo",
			expectedSubkeys: []string{"foo"},
			expectedFilters: map[int][]confdb.FieldFilter{},
		},
	}

	for _, tc := range tcs {
		cmt := Commentf("path: %q", tc.path)
		opts := confdb.ParseOptions{AllowPlaceholders: true}
		accessors, err := confdb.ParsePathIntoAccessors(tc.path, opts)

		if tc.err != "" {
			c.Check(err, ErrorMatches, tc.err, cmt)
			continue
		}

		c.Assert(err, IsNil, cmt)
		c.Assert(len(accessors), Equals, len(tc.expectedSubkeys), cmt)

		for i, acc := range accessors {
			c.Check(acc.Name(), Equals, tc.expectedSubkeys[i], cmt)

			// Check filters
			filters := confdb.GetFilters(acc)
			expectedFilters := tc.expectedFilters[i]
			if len(expectedFilters) == 0 {
				c.Check(len(filters), Equals, 0, cmt)
			} else {
				c.Assert(len(filters), Equals, len(expectedFilters), cmt)
				for j, filter := range filters {
					c.Check(filter.Field, Equals, expectedFilters[j].Field, cmt)
					c.Check(filter.Placeholder, Equals, expectedFilters[j].Placeholder, cmt)
				}
			}
		}
	}
}

func (*filtersSuite) TestFiltersInStoragePath(c *C) {
	// Test that filters can be used in storage paths and are preserved
	views := map[string]any{
		"test-view": map[string]any{
			"rules": []any{
				map[string]any{
					"request": "foo.{bar}",
					"storage": "data.{bar}[.type={bar}]",
				},
			},
		},
	}

	schema := confdb.NewJSONSchema()
	dbSchema, err := confdb.NewSchema("acc", "db", views, schema)
	c.Assert(err, IsNil)
	c.Assert(dbSchema, NotNil)

	view := dbSchema.View("test-view")
	c.Assert(view, NotNil)
}

func (*filtersSuite) TestFiltersDoNotAffectMatching(c *C) {
	// Test that filters don't affect the matching behavior
	views := map[string]any{
		"test-view": map[string]any{
			"rules": []any{
				map[string]any{
					"request": "foo.{bar}",
					"storage": "data.{bar}[.type={bar}]",
				},
			},
		},
	}

	schema := confdb.NewJSONSchema()
	dbSchema, err := confdb.NewSchema("acc", "db", views, schema)
	c.Assert(err, IsNil)

	view := dbSchema.View("test-view")
	c.Assert(view, NotNil)

	// Create a databag and test that Set/Get work normally
	databag := confdb.NewJSONDatabag()

	// Set a value - this should work regardless of filters
	err = view.Set(databag, "foo.test", "value123")
	c.Assert(err, IsNil)

	// Get the value back - filters shouldn't interfere
	result, err := view.Get(databag, "foo.test")
	c.Assert(err, IsNil)
	c.Check(result, Equals, "value123")
}

func (*filtersSuite) TestMultipleFiltersOnSameAccessor(c *C) {
	// Test multiple filters on the same accessor
	path := "foo[.a={b}][.c={d}][.e={f}].bar"
	opts := confdb.ParseOptions{AllowPlaceholders: true}
	accessors, err := confdb.ParsePathIntoAccessors(path, opts)
	c.Assert(err, IsNil)
	c.Assert(len(accessors), Equals, 2)

	// First accessor (foo) should have 3 filters
	filters := confdb.GetFilters(accessors[0])
	c.Assert(len(filters), Equals, 3)
	c.Check(filters[0].Field, Equals, "a")
	c.Check(filters[0].Placeholder, Equals, "b")
	c.Check(filters[1].Field, Equals, "c")
	c.Check(filters[1].Placeholder, Equals, "d")
	c.Check(filters[2].Field, Equals, "e")
	c.Check(filters[2].Placeholder, Equals, "f")

	// Second accessor (bar) should have no filters
	filters = confdb.GetFilters(accessors[1])
	c.Check(len(filters), Equals, 0)
}

func (*filtersSuite) TestFilterValidation(c *C) {
	// Test that filter field names and placeholders follow the same rules as subkeys
	type testcase struct {
		path string
		err  string
	}

	tcs := []testcase{
		{
			// Valid filter
			path: "foo[.valid-name={valid-placeholder}]",
			err:  "",
		},
		{
			// Valid filter with numbers
			path: "foo[.field123={place456}]",
			err:  "",
		},
		{
			// Not a valid filter (no dot), and not a valid index or placeholder either
			path: "foo[valid={placeholder}]",
			err:  `invalid subkey.*`, // Invalid format - neither filter, index, nor placeholder
		},
	}

	for _, tc := range tcs {
		cmt := Commentf("path: %q", tc.path)
		opts := confdb.ParseOptions{AllowPlaceholders: true}
		_, err := confdb.ParsePathIntoAccessors(tc.path, opts)

		if tc.err != "" {
			c.Check(err, ErrorMatches, tc.err, cmt)
		} else {
			c.Check(err, IsNil, cmt)
		}
	}
}

func (*filtersSuite) TestFiltersWithIndexPlaceholders(c *C) {
	// Test filters can coexist with index placeholders
	path := "foo[{idx}][.type={t}].bar"
	opts := confdb.ParseOptions{AllowPlaceholders: true}
	accessors, err := confdb.ParsePathIntoAccessors(path, opts)
	c.Assert(err, IsNil)
	c.Assert(len(accessors), Equals, 3) // foo, [{idx}], bar

	// Second accessor ([{idx}]) should have a filter
	filters := confdb.GetFilters(accessors[1])
	c.Assert(len(filters), Equals, 1)
	c.Check(filters[0].Field, Equals, "type")
	c.Check(filters[0].Placeholder, Equals, "t")
}

func (*filtersSuite) TestGetFiltersOnNonFilteredAccessor(c *C) {
	// Test that GetFilters returns nil for accessors without filters
	path := "foo.bar"
	opts := confdb.ParseOptions{AllowPlaceholders: true}
	accessors, err := confdb.ParsePathIntoAccessors(path, opts)
	c.Assert(err, IsNil)
	c.Assert(len(accessors), Equals, 2)

	filters := confdb.GetFilters(accessors[0])
	c.Check(filters, IsNil)

	filters = confdb.GetFilters(accessors[1])
	c.Check(filters, IsNil)
}
