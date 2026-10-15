# Schedule editor accepts input before loading finishes

During deployed browser verification, editing immediately after opening the form
could race the asynchronous schedule fetch. The submit button was disabled, but
other fields were editable and could be overwritten by the response. The first
automated edit did not navigate; waiting for loading and repeating succeeded.

Fix: disable the entire fieldset while fetching or saving and reset loading when
the schedule ID changes. Recheck with an intentionally delayed response.

Verified live after `make host-rebuild-markovd`: hold the GET response with a
Playwright route, open Edit, and inspect the fieldset before releasing it. The
fieldset is disabled. After release, editing, saving, pause/resume and delete pass.
