## Security Model

Users with passwords (and emails?).
Users will consult a special endpoint and receive an ephemeral key (think AWS)
This key maps onto a set of roles (think groups)
Each stream has a mapping of what roles are allowed to do what (public: read only, uc berkeley: read only, room 410: read + write, etc)
When you get an ephemeral key, you resolve the mapping onto the set of roles, and then cache
that for the duration of the key's life.

TODO: look at using OAuth to manage user accounts?

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
