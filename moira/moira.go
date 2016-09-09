package moira

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"

	"gopkg.in/ldap.v2"
)

// nfsgroup MUST match [a-z0-9-] (no LDAP quoting is done)
func GetMoiraNFSGroupMembers(nfsgroup string) ([]string, error) {
	l, err := ldap.DialTLS("tcp", "ldap.mit.edu:636", &tls.Config{ServerName: "ldap.mit.edu"})
	if err != nil {
		log.Print(err)
		return nil, err
	}
	defer l.Close()

	// ldapsearch -LLL -x -H ldap://ldap.mit.edu:389 -b "ou=lists,ou=moira,dc=mit,dc=edu" "cn=${nfsgroup}" member
	sr, err := l.Search(ldap.NewSearchRequest(
		"ou=lists,ou=moira,dc=mit,dc=edu",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases /*sizelimit*/, 0 /*timelimit*/, 0 /*typesonly*/, false,
		"(cn="+nfsgroup+")",
		[]string{"member"},
		/*"control"*/ nil,
	))
	if err != nil {
		log.Print(err)
		return nil, err
	}
	if l := len(sr.Entries); l != 1 {
		return nil, fmt.Errorf("expected exactly one list, found %d", l)
	}
	return sr.Entries[0].GetAttributeValues("member"), nil
}

func IsAuthorized(authorize string, email string) error {
	email = strings.ToLower(email)

	members, err := GetMoiraNFSGroupMembers(authorize)
	if err != nil {
		return err
	}

	// USER entries -- MIT kerberos accounts
	at_mit_edu := "@mit.edu"
	if strings.HasSuffix(email, at_mit_edu) && strings.Count(email, "@") == 1 {
		kerberos := strings.TrimSuffix(email, at_mit_edu)
		for _, member := range members {
			if member == "uid="+kerberos+",OU=users,OU=moira,dc=MIT,dc=EDU" {
				return nil
			}
		}
	}

	// STRING entries -- full email addresses
	for _, member := range members {
		if member == "cn="+email+",OU=strings,OU=moira,dc=MIT,dc=EDU" {
			return nil
		}
	}

	return fmt.Errorf("authenticated as %q, but not authorized because not on moira list %q", email, authorize)
}
