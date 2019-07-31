/*
Copyright © 2018 inwinSTACK Inc
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
   http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package operator

import (
	"context"
	"testing"

	blendedfake "github.com/inwinstack/blended/generated/clientset/versioned/fake"
	"github.com/inwinstack/ip-assigner/pkg/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
)

func TestOperator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Config{Threads: 2, SyncSec: 60}
	clientset := fake.NewSimpleClientset()
	blendedset := blendedfake.NewSimpleClientset()

	op := New(cfg, clientset, blendedset)
	assert.NotNil(t, op)
	assert.Nil(t, op.Run(ctx))

	cancel()
	op.Stop()
}
