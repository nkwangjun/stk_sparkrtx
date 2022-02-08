# Copyright 2020 Google LLC All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# build go
FROM golang:1.17.6 as go-builder
WORKDIR /go/src/supertuxkart

RUN apt-get update && apt-get install -y curl tar xz-utils

COPY main.go .
COPY api ./api
COPY grpcsdk ./grpcsdk
COPY gsemanager ./gsemanager
COPY logger ./logger
COPY go.mod .
RUN go mod tidy
RUN go build -o wrapper .


# final image
FROM gcr.io/agones-images/supertuxkart-example:0.4

WORKDIR /home/supertuxkart/stk-code

COPY --from=go-builder /go/src/supertuxkart/wrapper .

USER root

RUN chown -R supertuxkart:supertuxkart /home/supertuxkart && chmod +x wrapper

USER 1000
ENTRYPOINT ["./entrypoint.sh"]
