// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2024 Canonical Ltd
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

package confdb

type (
	ViewRef  = viewRef
	Accessor = accessor
)

var (
	GetValuesThroughPaths = getValuesThroughPaths
	NewAuthentication     = newAuthentication
	JoinAccessors         = joinAccessors
)

type Authentication = authentication

func (a Authentication) ToStrings() []string {
	return a.toStrings()
}

func PathValuePairsIntoMap(pairs []pathValuePair) map[string]any {
	pathsToValues := make(map[string]any, len(pairs))
	for _, pair := range pairs {
		pathsToValues[joinAccessors(pair.path)] = pair.value
	}

	return pathsToValues
}

func ParsePathIntoAccessors(path string) ([]accessor, error) {
	opts := parseOpts{allowPlaceholders: true}
	return parsePathIntoAccessors(path, opts)
}

func MockMaxValueDepth(newDepth int) (restore func()) {
	oldDepth := maxValueDepth
	maxValueDepth = newDepth
	return func() {
		maxValueDepth = oldDepth
	}
}
