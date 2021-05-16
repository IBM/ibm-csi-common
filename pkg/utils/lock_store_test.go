/**
 * Copyright 2020 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package utils ...
package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLockStoreFunctionalityCheck(t *testing.T) {
	var mutex LockStore
	mutex.Lock("TestLock1")
	defer mutex.Unlock("TestLock1")
	assert.Equal(t, 1, len(mutex.store)) // There will be only one lock

	mutex.Lock("TestLock2")
	//defer mutex.Unlock("TestLock2")
	assert.Equal(t, 2, len(mutex.store)) // There should be 2 lock

	// Re-locking with existing lock, it will wait till previous one unlock
	// hence unlocking previous one explicitly
	mutex.Unlock("TestLock2") // unlock it
	mutex.Lock("TestLock2")
	defer mutex.Unlock("TestLock2")
	assert.Equal(t, 2, len(mutex.store)) // It should have only 2 lock now as well
}
