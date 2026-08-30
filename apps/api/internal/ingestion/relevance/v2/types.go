package v2

// Experience level describes the target candidate seniority band.
type ExperienceLevel string

const (
	ExperienceInternship    ExperienceLevel = "internship"
	ExperienceCoOp          ExperienceLevel = "co_op"
	ExperienceNewGrad       ExperienceLevel = "new_grad"
	ExperienceEarlyCareer   ExperienceLevel = "early_career"
	ExperienceApprenticeship ExperienceLevel = "apprenticeship"
	ExperienceFellowship    ExperienceLevel = "fellowship"
	ExperienceUnknown       ExperienceLevel = "unknown"
)

// CareerFamily groups roles into technical career categories served by CareerOS.
type CareerFamily string

const (
	CareerSoftwareEngineering        CareerFamily = "software_engineering"
	CareerDataScience                CareerFamily = "data_science"
	CareerMachineLearningAI          CareerFamily = "machine_learning_ai"
	CareerCybersecurity              CareerFamily = "cybersecurity"
	CareerProductManagementTechnical CareerFamily = "product_management_technical"
	CareerCloudInfrastructureDevOps  CareerFamily = "cloud_infrastructure_devops"
	CareerQuantitativeTechnology     CareerFamily = "quantitative_technology"
	CareerTechnicalResearch          CareerFamily = "technical_research"
	CareerOtherTechnical             CareerFamily = "other_technical"
	CareerNonTechnical               CareerFamily = "non_technical"
	CareerUnknown                    CareerFamily = "unknown"
)

// EducationLevel captures detected education requirements when present.
type EducationLevel string

const (
	EducationUndergraduate EducationLevel = "undergraduate"
	EducationMasters       EducationLevel = "masters"
	EducationPhD           EducationLevel = "phd"
	EducationGraduateAny   EducationLevel = "graduate_any"
	EducationUnspecified   EducationLevel = "unspecified"
)

// RelevanceTier drives product feed inclusion policy.
type RelevanceTier string

const (
	TierHighConfidenceTechnical    RelevanceTier = "high_confidence_technical"
	TierAmbiguous                  RelevanceTier = "ambiguous"
	TierHighConfidenceNonTechnical RelevanceTier = "high_confidence_non_technical"
)

// Classification is the full deterministic relevance output for a posting.
type Classification struct {
	ExperienceLevel ExperienceLevel
	CareerFamily    CareerFamily
	EducationLevel  EducationLevel
	RelevanceTier   RelevanceTier
	Reasons         []string
	InTechnicalFeed bool
}
