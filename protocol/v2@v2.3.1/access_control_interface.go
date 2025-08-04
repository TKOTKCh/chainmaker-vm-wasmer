/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker/common/v2/crypto"
	pbac "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/config"
)

// 以下ac模块相关常量
const (
	// ConfigNameOrgId org_id
	ConfigNameOrgId = "org_id"
	// ConfigNameRoot root
	ConfigNameRoot = "root"
	// CertFreezeKey CERT_FREEZE
	CertFreezeKey = "CERT_FREEZE"
	//CertFreezeKeyPrefix freeze_
	CertFreezeKeyPrefix = "freeze_"
	//CertRevokeKey CERT_CRL
	CertRevokeKey = "CERT_CRL"
	//CertRevokeKeyPrefix c_
	CertRevokeKeyPrefix = "c_"

	// fine-grained resource id for different policies

	ResourceNameUnknown          = "UNKNOWN"
	ResourceNameReadData         = "READ"
	ResourceNameWriteData        = "WRITE"
	ResourceNameP2p              = "P2P"
	ResourceNameConsensusNode    = "CONSENSUS"
	ResourceNameAdmin            = "ADMIN"
	ResourceNameUpdateConfig     = "CONFIG"
	ResourceNameUpdateSelfConfig = "SELF_CONFIG"
	ResourceNameAllTest          = "ALL_TEST"

	ResourceNameTxQuery    = "query"
	ResourceNameTxTransact = "transaction"

	ResourceNamePrivateCompute = "PRIVATE_COMPUTE"
	ResourceNameArchive        = "ARCHIVE"
	ResourceNameSubscribe      = "SUBSCRIBE"

	RoleAdmin         Role = "ADMIN"
	RoleClient        Role = "CLIENT"
	RoleLight         Role = "LIGHT"
	RoleConsensusNode Role = "CONSENSUS"
	RoleCommonNode    Role = "COMMON"
	RoleContract      Role = "CONTRACT"

	RuleMajority  Rule = "MAJORITY"
	RuleAll       Rule = "ALL"
	RuleAny       Rule = "ANY"
	RuleSelf      Rule = "SELF"
	RuleForbidden Rule = "FORBIDDEN"
	RuleDelete    Rule = "DELETE"
)

// AuthType authorization type
type AuthType uint32

const (
	//PermissionedWithCert permissioned with certificate
	PermissionedWithCert string = "permissionedwithcert"

	//PermissionedWithKey permissioned with public key
	PermissionedWithKey string = "permissionedwithkey"

	// Public public key
	Public string = "public"

	// Identity (1.X PermissionedWithCert)
	Identity string = "identity"
)

// Role for members in an organization
type Role string

// Rule Keywords of authentication rules
type Rule string

// Principal contains all information related to one time verification
type Principal interface {
	// GetResourceName returns resource name of the verification
	GetResourceName() string

	// GetEndorsement returns all endorsements (signatures) of the verification
	GetEndorsement() []*common.EndorsementEntry

	// GetMessage returns signing data of the verification
	GetMessage() []byte

	// GetTargetOrgId returns target organization id of the verification if the verification is for a specific organization
	GetTargetOrgId() string
}

// AccessControlProvider manages policies and principals.
type AccessControlProvider interface {

	// GetHashAlg return hash algorithm the access control provider uses
	GetHashAlg() string

	// ValidateResourcePolicy checks whether the given resource policy is valid
	ValidateResourcePolicy(resourcePolicy *config.ResourcePolicy) bool

	// LookUpPolicy returns corresponding policy configured for the given resource name
	LookUpPolicy(resourceName string) (*pbac.Policy, error)

	// LookUpExceptionalPolicy returns corresponding exceptional policy configured for the given resource name
	LookUpExceptionalPolicy(resourceName string) (*pbac.Policy, error)

	//GetAllPolicy returns all policies
	GetAllPolicy() (map[string]*pbac.Policy, error)

	// CreatePrincipal creates a principal for one time authentication
	CreatePrincipal(resourceName string, endorsements []*common.EndorsementEntry, message []byte) (Principal, error)

	// CreatePrincipalForTargetOrg creates a principal for "SELF" type policy,
	// which needs to convert SELF to a sepecific organization id in one authentication
	CreatePrincipalForTargetOrg(resourceName string, endorsements []*common.EndorsementEntry, message []byte,
		targetOrgId string) (Principal, error)

	//GetValidEndorsements filters all endorsement entries and returns all valid ones
	GetValidEndorsements(principal Principal) ([]*common.EndorsementEntry, error)

	// VerifyPrincipal verifies if the policy for the resource is met
	VerifyPrincipal(principal Principal) (bool, error)

	// NewMember creates a member from pb Member
	NewMember(member *pbac.Member) (Member, error)

	//GetMemberStatus get the status information of the member
	GetMemberStatus(member *pbac.Member) (pbac.MemberStatus, error)

	//VerifyRelatedMaterial verify the member's relevant identity material
	VerifyRelatedMaterial(verifyType pbac.VerifyType, data []byte) (bool, error)
}

// Member is the identity of a node or user.
type Member interface {
	// GetMemberId returns the identity of this member (non-uniqueness)
	GetMemberId() string

	// GetOrgId returns the organization id which this member belongs to
	GetOrgId() string

	// GetRole returns roles of this member
	GetRole() Role

	// GetUid returns the identity of this member (unique)
	GetUid() string

	// Verify verifies a signature over some message using this member
	Verify(hashType string, msg []byte, sig []byte) error

	// GetMember returns Member
	GetMember() (*pbac.Member, error)

	//GetPk returns public key
	GetPk() crypto.PublicKey
}

// SigningMember signer
type SigningMember interface {
	// Extends Member interface
	Member

	// Sign signs the message with the given hash type and returns signature bytes
	Sign(hashType string, msg []byte) ([]byte, error)
}
