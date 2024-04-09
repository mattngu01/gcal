# Google Calendar TUI

Created using [Google's quickstart](https://developers.google.com/calendar/api/quickstart/go). Built as a personal project to learn Go.

## Getting using user credentials

Follow the [steps here](https://developers.google.com/calendar/api/quickstart/go#authorize_credentials_for_a_desktop_application) to save the OAuth Client credentials as `credentials.json` in the same working folder.

## Contributing

Commits should follow the [Angular changelog convention](https://github.com/conventional-changelog/conventional-changelog/tree/master/packages/conventional-changelog-angular) header format. 


## Build pipeline

Pipeline uses [GH Release](https://github.com/marketplace/actions/gh-release) and [Github Tag](https://github.com/marketplace/actions/github-tag) actions.

## TODO
- [X] new event form
- [ ] customize date output
- [ ] prettier detailed view
- [ ] improve token initialization flow
- [X] delete events
- [X] edit events
- [X] refresh calendar
- [X] build pipeline