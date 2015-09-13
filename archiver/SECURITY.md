## Security Model

Users with passwords (and emails?).
Users will consult a special endpoint and receive an ephemeral key (think AWS)
This key maps onto a set of roles (think groups)
Each stream has a mapping of what roles are allowed to do what (public: read only, uc berkeley: read only, room 410: read + write, etc)
When you get an ephemeral key, you resolve the mapping onto the set of roles, and then cache
that for the duration of the key's life.

TODO: look at using OAuth to manage user accounts?

### Process

1. How do I secure a stream (the principal)?
There are 3 types of permissions with relation to a stream:
* PUBLISH (write data to the archiver as the stream)
* READ (can read data and metadata pertaining to the stream from the archiver)
* EXECUTE (can forward data directly to the stream?)

Maybe we don't have push/execute. Instead, if a stream *wants* to receive messages, it
can create a a special subscription that the publishers must know.  In that
case, we will probably need a way to discover what those keys are and that
permission will take the place of this one.

Then, I need to establish what the permitted permission are for each of the roles
that might access. Permissions can be allocated for GLOBAL or for GROUP

### Notes

AES encryption for payloads? This is symmetric, which works for us. Archiver has a different symmetric key for each stream.
Revocation for a stream is easy then, because then you can just get rid of the other half of the symmetric key.
Archiver needs to be able to read all data so it can do routing.
Assumption: broker is able to read everything.
Archiver is TRUSTED. use something like bosswave to create WANs from multiple archivers

Archiver has some known certificate used to secure communication?

This is fine bc you are operating w/n a domain of trust. Archiver is an arbiter
for distribution and storage of information.  publishers and subscribers should
be hidden from each other. archiver bridges physical networks as well as
administrative domains (eh)

STREAMs belong to a SOURCE. A SOURCE can be thought of as some authority for
some domain of data (e.g. a whole sensor network would probably be a SOURCE, as
would a lighting controller or an individual thermostat). A SOURCE is manually
registered w/ the archiver through some out-of-band method, e.g. logging in
via a user SSH account (bound to Unix permissions?) and registering a new source.

When a SOURCE is created, it receives a private key from the archiver. When a STREAM
is created for a SOURCE, the private key from the SOURCE is used to generate the symmetric
key used for that stream. This symmetric key is sent to the archiver to register the stream,
and should somehow be signed by the SOURCE so the archiver can ensure that the symmetric
key actually belongs to that SOURCE.

SOURCES will AES encrypt all messages to the archiver with their given symmetric key. The archiver
also has the symmetric key, and thus will be able to read all messages. If a symmetric key is removed
from the archiver, this is essentially revoking the stream. All messages delivered TO the stream
will also be AES encrypted w/ the same key.
I think SOURCES should be able to transmit/receive data "in the clear" if they choose. Sometimes this
is very helpful. This goes in line with the GLOBAL permissions (e.g. GLOBAL reado nly, GLOBAL no encrypt, etc).

Okay so we *do* have user accounts, which are not necessarily individuals, but could also
be administrative groups or arbitrary domain. We have users as the point of contact between
the human and the rest of the system. There are 2 types of users: admin and normal. Admins
can add/remove roles and can create/remove streams everywhere. Basically free reign of the system (for now).
Users can only administrate stuff for the SOURCES that they have.

ROLEs are attached to USERs. A ROLE is some named entity created by an ADMIN
and maps to Pub/read/exec permissions on a stream by stream basis. It is up to
the SOURCE (or an ADMIN) for a stream to determine what ROLES have what
permissions on that stream. If a ROLE is not listed, it obviously doesn't have
any permissions.

STREAMS and ROLES: Each stream has a mapping of (ROLE -> permissions). But how does that work?
This has to be FAST.

How do we interact w/ the archiver as applications and/or users?
Login to the archiver with your accoutn, create an ephemeral key with some (user-specified) expiry.
This key gets mapped internally to a set of roles to speed up the lookups (and to avoid having to
go look at the user account each time). This key is somehow included whenever the user interacts. Is this
key used for symmetric encryption? Why not! That's good.

need to look into FAST symmetric encryption in go.


# 1. How do I create a user account?

User accounts should be established out-of-band
Roles and their permissions are established out-of-band?


### Interfaces

Also want to be able to remain agnostic to the underlying datastore (we are hopefully moving to a relational store eventually), so let's
define the interface.

Account Manager
func CreateUser(user Username, email Email, password Password) -> User (remember to hash the password!)
func (u User) AddRole(role Role) error
func (u User) DelRole(role Role) error
func (u User) ListRoles() error

for this, basically just have a big lookup table, making sure to have 1-N relationship for User-Roles

Key/Role/Stream Manager

get ephemeral keys
resolve ephemeral key to a set of roles and cache that for lookup

func GetEphemeralKey(user Username, password Password, valid Duration) -> EphemeralKey

need to add/remove roles and permissions to a stream. Also need to cache permissions for the ephemeral key? not sure here
