# gocd-jira-release-notes

HTTP service which gets all commits from a GoCD pipeline build, finds Release Notes in related JIRA issues and publishes the aggregated Release Notes as Blog posts in Confluence.

## Why

As an Iotics Product Owner
I want to provide release notes for every release of our Product
So that I can inform our customers of upcoming features, improvements, breaking changes and bug fixes.

## How

This is built as a service, so that it can be triggered ad-hoc
and so that the rather complex functionality is wrapped in a tested and self-documented code.

The service is triggered at the end of our deployment pipeline, passing in the desired Blog `title` and GoCD `pipeline` name and the build `counter`.

E.g.

```bash
curl -k <serviceUri>?title=OurProject&pipeline=iotic-service&counter=99
```

Remark: This whole orchestration would be possible in a bash script. It probably would be complex and not easily testable.

## What

Steps:

- This service runs as a _Function as a Service_ (using _OpenFaaS_) on our internal infrastructure.
- GoCD pipeline triggers this function, passing in a pipeline name and a pipeline counter; this function then:
- Calls GoCD API to get the pipeline details (label aka version)
- Calls GoCD API to get a comparison of the pipeline to the previous version of the pipeline (to get all changes/commits in this version)
- Parses the commit messages and finds JIRA issue prefixes in them
- Calls JIRA API to get details of the JIRA issues (specifically a custom field which contains "Release Notes")
- Aggregates the release notes by the headings
- Converts the release notes to HTML/markup format used by Confluence
- Publishes the release notes to Confluence as a blog post. The blog post has a label with the name of the pipeline.

## Pre-requisites

- GoCD API token
- JIRA/Confluence API token
- Necessary permissions to read from GoCD and JIRA and to write to Confluence

## Secrets

There are two secrets which need to be set:

- `gocdApikey`
- `jiraApikey`

During runtime, these can be read from a k8s/OpenFaaS secret, which should be automatically mounted as `/var/openfaas/secrets/<secret>`.

During development, please update a local `config.yaml` accordingly.

NOTE: The `jiraApiKey` is shared by both JIRA and Confluence.

## Testing

The `./examples` directory contains sample JSON data from GoCD and JIRA. These are used by some tests.

There is a flag in the `server_test.go` which allows to make use of real responses when it's set to `true`. This is great for running tests and troubleshooting when e.g. GoCD or JIRA/Confluence API changes, and to actually see an end result in Confluence.

However for unit testing purposes, this flag is set to `false` so that the stored JSON data is used instead.

See `var useMockedResponse = true`

## QnA

Q: Why using REST API's directly and not some clients in Go? There are clients in Go for GoCD, JIRA and Confluence.

A: Because I haven't found any client in Go which would look active and maintained.
A: Also because of additional dependencies on potentially questionable 3rd parties.

- <https://github.com/ctreminiom/go-atlassian> is new, but too new, Confluence API is not implemented.
- Specific JIRA or Confluence clients in Go are either years old, not maintained, or work with on-prem installed products and not with Cloud.

## Notes

It would be possible to optimize the request to get a JIRA issue by stripping down unnecessary information.

E.g. <https://ioticlabs.atlassian.net/rest/agile/latest/issue/FO-1949?fields=-summary,-comment,-description,-issuelinks,-project,-watches,-worklog,-watches,-votes,-reporter,-subtasks,-creator,-priority,-closedSprints&expand=names&properties=-self>

It would probably make a tiny bit faster but maybe more difficult to maintain and test/mock.

For the time being, no such optimization is done, the full JIRA issue details are requested and parsed.

## Useful Links

### GoCD

- <https://api.gocd.org/21.1.0/#get-pipeline-instance>
- <https://github.com/gocd/gocd-filebased-authentication-plugin#readme>

### JIRA/Confluence

- <https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issues/#api-rest-api-3-issue-createmeta-get>
- <https://developer.atlassian.com/cloud/confluence/rest/api-group-content/#api-api-content-post>
- <https://id.atlassian.com/manage-profile/security/api-tokens>
- <https://community.atlassian.com/t5/Answers-Developer-Questions/Creating-a-confluence-page-via-rest-api-with-a-label/qaq-p/469849>

## Technology Used

- Confluence
- REST API
- OpenFaaS
- Docker
- Shell
- JIRA
- GoCD
- JSON
- Git
- Go

```

```
