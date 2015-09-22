## Subscription Initialization

HTTP: Send a POST request to /api/subscribe containing the query you want to subscribe to


You can only subscribe to "select" queries, but these can be augmented with operators

When you instigate a subscription, you are first delivered the results of your query and then
continue to receive updates

I think we can do even more selective reevaluations. We have "where" tags and "select" tags
When a where tag changes:
    could change the range of streams that qualify, so we re-run the
    "where" and then the full query for these

When a select tag changes:
    the set of qualified streams has NOT changed, but the answer to our query might.
    Maybe we use some kind of embedded store to fix the mappings of these at a higher level?
    stream -- query -- key -- value relation? When select key [x] changes, we
    can alter only the in-memory representation in the hotpath, and then save to the db
    in the background

select distinct Metadata/HVACZone [where XYZ]:
    Updates: rerun the whole query when it might have changed
    New stream: rerun the whole query
    Del stream: rerun the whole query

select tag1, tag2 [where XYZ]:
    Updates: Deliver the message that changed (["uuid": "...", "Metadata": {"tag1": "new value"}])
    New: deliver just that message
    Del: deliver just that uuid

select data before now [where XYZ]
