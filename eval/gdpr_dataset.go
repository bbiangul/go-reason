package eval

// GDPREasyDataset returns 30 easy (single-fact lookup) test cases
// from the GDPR (Regulation (EU) 2016/679).
//
// Expected facts use pipe-separated alternatives so accuracy scoring
// works regardless of the LLM's paraphrasing or formatting choices.
func GDPREasyDataset() Dataset {
	return Dataset{
		Name:       "GDPR Easy - Single Fact Lookup",
		Difficulty: DifficultyEasy,
		Tests: []TestCase{
			{
				Question:      "What is the full name of the regulation known as the GDPR?",
				ExpectedFacts: []string{"General Data Protection Regulation|Regulation (EU) 2016/679"},
				Category:      "single-fact",
				Explanation:   "Title page: 'Regulation (EU) 2016/679 of the European Parliament and of the Council'.",
			},
			{
				Question:      "When did the GDPR enter into application?",
				ExpectedFacts: []string{"25 May 2018|25/05/2018"},
				Category:      "single-fact",
				Explanation:   "Art. 99(2): 'It shall apply from 25 May 2018.'",
			},
			{
				Question:      "What does 'personal data' mean under GDPR?",
				ExpectedFacts: []string{"information relating to an identified or identifiable natural person|data subject"},
				Category:      "single-fact",
				Explanation:   "Art. 4(1) definition.",
			},
			{
				Question:      "What is the definition of 'consent' in the GDPR?",
				ExpectedFacts: []string{"freely given, specific, informed and unambiguous|indication of the data subject's wishes"},
				Category:      "single-fact",
				Explanation:   "Art. 4(11).",
			},
			{
				Question:      "What age threshold does GDPR set for children's consent for information society services?",
				ExpectedFacts: []string{"16 years|16"},
				Category:      "single-fact",
				Explanation:   "Art. 8(1): 'the child is at least 16 years old'.",
			},
			{
				Question:      "What is the maximum administrative fine for the most serious GDPR infringements?",
				ExpectedFacts: []string{"20 000 000|20000000|4 % of the total worldwide annual turnover|4%"},
				Category:      "single-fact",
				Explanation:   "Art. 83(5).",
			},
			{
				Question:      "What body serves as the EU-level supervisory authority under GDPR?",
				ExpectedFacts: []string{"European Data Protection Board|EDPB|Board"},
				Category:      "single-fact",
				Explanation:   "Art. 68 establishes the Board.",
			},
			{
				Question:      "What is the definition of 'controller' in the GDPR?",
				ExpectedFacts: []string{"natural or legal person|determines the purposes and means of the processing"},
				Category:      "single-fact",
				Explanation:   "Art. 4(7).",
			},
			{
				Question:      "What is a 'Data Protection Impact Assessment'?",
				ExpectedFacts: []string{"assessment of the impact|processing operations on the protection of personal data|DPIA"},
				Category:      "single-fact",
				Explanation:   "Art. 35(1).",
			},
			{
				Question:      "How long does a controller have to notify the supervisory authority of a personal data breach?",
				ExpectedFacts: []string{"72 hours|without undue delay|3 days"},
				Category:      "single-fact",
				Explanation:   "Art. 33(1).",
			},
			{
				Question:      "What does 'processing' mean under GDPR?",
				ExpectedFacts: []string{"operation|collection, recording, organisation, structuring|set of operations"},
				Category:      "single-fact",
				Explanation:   "Art. 4(2).",
			},
			{
				Question:      "What is the definition of 'processor' under GDPR?",
				ExpectedFacts: []string{"natural or legal person|processes personal data on behalf of the controller"},
				Category:      "single-fact",
				Explanation:   "Art. 4(8).",
			},
			{
				Question:      "What is 'pseudonymisation' as defined by the GDPR?",
				ExpectedFacts: []string{"processing of personal data|no longer be attributed to a specific data subject without the use of additional information"},
				Category:      "single-fact",
				Explanation:   "Art. 4(5).",
			},
			{
				Question:      "What does 'profiling' mean under the GDPR?",
				ExpectedFacts: []string{"automated processing of personal data|evaluating certain personal aspects|analyse or predict"},
				Category:      "single-fact",
				Explanation:   "Art. 4(4).",
			},
			{
				Question:      "What is a 'binding corporate rule' under GDPR?",
				ExpectedFacts: []string{"personal data protection policies|adhered to by a controller or processor|group of undertakings"},
				Category:      "single-fact",
				Explanation:   "Art. 4(20).",
			},
			{
				Question:      "What is the 'right to erasure' also known as?",
				ExpectedFacts: []string{"right to be forgotten|erasure"},
				Category:      "single-fact",
				Explanation:   "Art. 17 title: \"Right to erasure ('right to be forgotten')\".",
			},
			{
				Question:      "When was the GDPR officially adopted (date of the regulation)?",
				ExpectedFacts: []string{"27 April 2016|27/04/2016"},
				Category:      "single-fact",
				Explanation:   "Header: 'of 27 April 2016'.",
			},
			{
				Question:      "Which regulation does the GDPR repeal?",
				ExpectedFacts: []string{"Directive 95/46/EC|95/46"},
				Category:      "single-fact",
				Explanation:   "Art. 94(1): 'Directive 95/46/EC is repealed'.",
			},
			{
				Question:      "What is the definition of 'recipient' under GDPR?",
				ExpectedFacts: []string{"natural or legal person|to which the personal data are disclosed"},
				Category:      "single-fact",
				Explanation:   "Art. 4(9).",
			},
			{
				Question:      "What is a 'supervisory authority' under GDPR?",
				ExpectedFacts: []string{"independent public authority|established by a Member State"},
				Category:      "single-fact",
				Explanation:   "Art. 4(21).",
			},
			{
				Question:      "What does 'filing system' mean in GDPR?",
				ExpectedFacts: []string{"structured set of personal data|accessible according to specific criteria"},
				Category:      "single-fact",
				Explanation:   "Art. 4(6).",
			},
			{
				Question:      "How many chapters does the GDPR contain?",
				ExpectedFacts: []string{"11|eleven"},
				Category:      "single-fact",
				Explanation:   "GDPR structure: Chapters I through XI.",
			},
			{
				Question:      "What is the principle of 'data minimisation'?",
				ExpectedFacts: []string{"adequate, relevant and limited to what is necessary|minimisation"},
				Category:      "single-fact",
				Explanation:   "Art. 5(1)(c).",
			},
			{
				Question:      "What does 'purpose limitation' mean under GDPR?",
				ExpectedFacts: []string{"collected for specified, explicit and legitimate purposes|not further processed in a manner that is incompatible"},
				Category:      "single-fact",
				Explanation:   "Art. 5(1)(b).",
			},
			{
				Question:      "What is the territorial scope of the GDPR?",
				ExpectedFacts: []string{"establishment of a controller or processor in the Union|offering of goods or services|monitoring of their behaviour", "Article 3"},
				Category:      "single-fact",
				Explanation:   "Art. 3.",
			},
			{
				Question:      "What is a 'personal data breach' under GDPR?",
				ExpectedFacts: []string{"breach of security|accidental or unlawful destruction, loss, alteration, unauthorised disclosure"},
				Category:      "single-fact",
				Explanation:   "Art. 4(12).",
			},
			{
				Question:      "Who is a 'Data Protection Officer'?",
				ExpectedFacts: []string{"DPO|data protection officer", "controller|processor"},
				Category:      "single-fact",
				Explanation:   "Art. 37(1).",
			},
			{
				Question:      "What information must be provided when personal data is collected from the data subject?",
				ExpectedFacts: []string{"identity and the contact details of the controller|purposes of the processing|legal basis"},
				Category:      "single-fact",
				Explanation:   "Art. 13(1).",
			},
			{
				Question:      "What does 'storage limitation' mean under GDPR?",
				ExpectedFacts: []string{"kept in a form which permits identification of data subjects for no longer than is necessary"},
				Category:      "single-fact",
				Explanation:   "Art. 5(1)(e).",
			},
			{
				Question:      "What is the 'right to data portability'?",
				ExpectedFacts: []string{"receive the personal data|structured, commonly used and machine-readable format|transmit those data"},
				Category:      "single-fact",
				Explanation:   "Art. 20(1).",
			},
		},
	}
}

// GDPRMediumDataset returns 30 medium (multi-hop / cross-article) test cases
// requiring synthesis across multiple GDPR articles.
func GDPRMediumDataset() Dataset {
	return Dataset{
		Name:       "GDPR Medium - Multi-hop / Cross-Article",
		Difficulty: DifficultyMedium,
		Tests: []TestCase{
			{
				Question:      "What are all the lawful bases for processing personal data, and where are they defined?",
				ExpectedFacts: []string{"consent", "contract", "legal obligation", "vital interests", "public interest|public task", "legitimate interests", "Article 6"},
				Category:      "multi-hop",
				Explanation:   "Art. 6(1)(a)-(f) lists all six lawful bases.",
			},
			{
				Question:      "How do the GDPR's data protection principles in Article 5 relate to the accountability obligation?",
				ExpectedFacts: []string{"lawfulness|fairness|transparency", "purpose limitation", "data minimisation", "accuracy", "storage limitation", "integrity|confidentiality", "accountability|demonstrate compliance"},
				Category:      "multi-hop",
				Explanation:   "Art. 5(1)(a)-(f) + Art. 5(2) accountability.",
			},
			{
				Question:      "What rights does a data subject have when subjected to automated decision-making including profiling?",
				ExpectedFacts: []string{"right not to be subject to a decision based solely on automated processing|right to obtain human intervention|express his or her point of view|contest the decision"},
				Category:      "multi-hop",
				Explanation:   "Art. 22(1) + Art. 22(3).",
			},
			{
				Question:      "What conditions must be met for valid consent under GDPR, and can it be withdrawn?",
				ExpectedFacts: []string{"freely given|specific|informed|unambiguous", "right to withdraw|at any time|as easy to withdraw as to give"},
				Category:      "multi-hop",
				Explanation:   "Art. 4(11) + Art. 7(1)-(3).",
			},
			{
				Question:      "How does GDPR connect Data Protection by Design and by Default to the controller's obligations?",
				ExpectedFacts: []string{"appropriate technical and organisational measures|implement data-protection principles|by design|by default", "state of the art|cost of implementation"},
				Category:      "multi-hop",
				Explanation:   "Art. 25(1)-(2).",
			},
			{
				Question:      "What is the relationship between the right to rectification and the right to erasure?",
				ExpectedFacts: []string{"rectification|erasure", "inaccurate|right to be forgotten|without undue delay", "Article 16|Article 17"},
				Category:      "multi-hop",
				Explanation:   "Art. 16 (rectification) references incomplete data; Art. 17 references erasure conditions.",
			},
			{
				Question:      "What exemptions does GDPR provide for processing special categories of personal data?",
				ExpectedFacts: []string{"explicit consent", "employment|social security|social protection", "vital interests", "legal claims|judicial acts", "substantial public interest|public interest", "health|medical|healthcare", "Article 9"},
				Category:      "multi-hop",
				Explanation:   "Art. 9(2)(a)-(j).",
			},
			{
				Question:      "How does the one-stop-shop mechanism work under the GDPR?",
				ExpectedFacts: []string{"lead supervisory authority|main establishment|cross-border processing|competent supervisory authority", "Article 56"},
				Category:      "multi-hop",
				Explanation:   "Art. 56(1)-(2) lead authority mechanism.",
			},
			{
				Question:      "What must a Data Protection Impact Assessment contain according to the GDPR?",
				ExpectedFacts: []string{"systematic description of the processing|assessment of the necessity and proportionality|assessment of the risks|measures envisaged to address the risks", "Article 35"},
				Category:      "multi-hop",
				Explanation:   "Art. 35(7)(a)-(d).",
			},
			{
				Question:      "What are the conditions for transferring personal data to a third country, and how do they relate to adequacy decisions?",
				ExpectedFacts: []string{"adequacy decision|Commission", "appropriate safeguards|binding corporate rules|standard data protection clauses", "Article 45|Article 46"},
				Category:      "multi-hop",
				Explanation:   "Art. 45 (adequacy) + Art. 46 (safeguards).",
			},
			{
				Question:      "How does GDPR handle the relationship between controllers and processors?",
				ExpectedFacts: []string{"contract|binding legal act", "subject-matter and duration|nature and purpose", "processor shall not engage another processor without|written authorisation", "Article 28"},
				Category:      "multi-hop",
				Explanation:   "Art. 28(1)-(3).",
			},
			{
				Question:      "What are the tasks of the European Data Protection Board?",
				ExpectedFacts: []string{"ensure the consistent application|advise the Commission|issue guidelines|accreditation of certification bodies|consistency mechanism", "Article 70"},
				Category:      "multi-hop",
				Explanation:   "Art. 70(1) lists Board tasks.",
			},
			{
				Question:      "How do the notification requirements differ between notifying the supervisory authority and the data subject of a breach?",
				ExpectedFacts: []string{"supervisory authority|72 hours", "data subject|without undue delay|high risk", "Article 33|Article 34"},
				Category:      "multi-hop",
				Explanation:   "Art. 33 (authority) vs Art. 34 (data subject — only if high risk).",
			},
			{
				Question:      "What restrictions does GDPR place on automated individual decision-making, and what exceptions exist?",
				ExpectedFacts: []string{"solely on automated processing|profiling|which produces legal effects|similarly significantly affects", "necessary for entering into or performance of a contract|authorised by Union or Member State law|explicit consent", "Article 22"},
				Category:      "multi-hop",
				Explanation:   "Art. 22(1)-(2).",
			},
			{
				Question:      "What is the relationship between codes of conduct and certification under GDPR?",
				ExpectedFacts: []string{"codes of conduct|associations and other bodies|monitoring compliance", "certification mechanisms|demonstrate compliance", "Article 40|Article 42"},
				Category:      "multi-hop",
				Explanation:   "Art. 40 (codes) + Art. 42 (certification) both serve as compliance tools.",
			},
			{
				Question:      "How does GDPR define and regulate 'joint controllers'?",
				ExpectedFacts: []string{"two or more controllers jointly determine|arrangement between them|respective responsibilities|essence of the arrangement|available to the data subject", "Article 26"},
				Category:      "multi-hop",
				Explanation:   "Art. 26(1)-(3).",
			},
			{
				Question:      "What are the specific obligations of a processor under GDPR?",
				ExpectedFacts: []string{"written instructions|instructions of the controller", "confidentiality", "security measures|security of processing", "sub-processor|another processor", "assist the controller", "delete or return", "Article 28"},
				Category:      "multi-hop",
				Explanation:   "Art. 28(3)(a)-(h).",
			},
			{
				Question:      "What role does the DPO play and when must one be designated?",
				ExpectedFacts: []string{"public authority|core activities|regular and systematic monitoring|large scale|special categories", "inform and advise|monitor compliance|contact point", "Article 37|Article 39"},
				Category:      "multi-hop",
				Explanation:   "Art. 37(1) (when) + Art. 39(1) (tasks).",
			},
			{
				Question:      "What is the consistency mechanism and how does it relate to the Board?",
				ExpectedFacts: []string{"consistency mechanism|opinion of the Board|relevant supervisory authority|binding decision", "Article 63|Article 64|Article 65"},
				Category:      "multi-hop",
				Explanation:   "Art. 63-65 define the mechanism.",
			},
			{
				Question:      "How does the GDPR regulate international transfers using standard contractual clauses?",
				ExpectedFacts: []string{"standard data protection clauses|adopted by the Commission|adopted by a supervisory authority and approved by the Commission|appropriate safeguards", "Article 46"},
				Category:      "multi-hop",
				Explanation:   "Art. 46(2)(c)-(d).",
			},
			{
				Question:      "How do the right to restriction of processing and the right to object relate?",
				ExpectedFacts: []string{"restriction of processing|right to object|pending verification|legitimate grounds", "Article 18|Article 21"},
				Category:      "multi-hop",
				Explanation:   "Art. 18(1)(d) links to Art. 21 objection.",
			},
			{
				Question:      "What remedies are available under GDPR and who can exercise them?",
				ExpectedFacts: []string{"right to lodge a complaint|right to an effective judicial remedy|right to compensation", "supervisory authority|controller|processor", "Article 77|Article 78|Article 79|Article 82"},
				Category:      "multi-hop",
				Explanation:   "Arts. 77-79 + 82.",
			},
			{
				Question:      "Under what conditions can a member state lower the age of consent for children below 16?",
				ExpectedFacts: []string{"Member States may provide by law for a lower age|not below 13 years", "Article 8"},
				Category:      "multi-hop",
				Explanation:   "Art. 8(1): 'Member States may provide by law for a lower age for those purposes provided that such lower age is not below 13 years'.",
			},
			{
				Question:      "What are the conditions under which a DPIA is mandatory?",
				ExpectedFacts: []string{"systematic and extensive evaluation|automated processing|profiling|large scale processing|special categories|publicly accessible areas", "Article 35"},
				Category:      "multi-hop",
				Explanation:   "Art. 35(3)(a)-(c).",
			},
			{
				Question:      "How does GDPR address data processing for scientific research and statistical purposes?",
				ExpectedFacts: []string{"archiving purposes in the public interest|scientific or historical research purposes|statistical purposes", "appropriate safeguards|technical and organisational measures", "Article 89"},
				Category:      "multi-hop",
				Explanation:   "Art. 89(1)-(2) + recitals on research exemptions.",
			},
			{
				Question:      "What is the relationship between a supervisory authority's investigative powers and corrective powers?",
				ExpectedFacts: []string{"investigative powers|corrective powers", "order the controller|impose administrative fine|advisory powers", "Article 58"},
				Category:      "multi-hop",
				Explanation:   "Art. 58(1) investigative + Art. 58(2) corrective.",
			},
			{
				Question:      "How does the GDPR address the processing of personal data relating to criminal convictions?",
				ExpectedFacts: []string{"criminal convictions and offences|official authority", "Article 10", "comprehensive register|Union or Member State law"},
				Category:      "multi-hop",
				Explanation:   "Art. 10.",
			},
			{
				Question:      "What obligations does a controller have regarding the right of access by the data subject?",
				ExpectedFacts: []string{"confirm whether or not personal data|provide a copy|purposes of the processing|categories of personal data|recipients|envisaged period|right to lodge a complaint", "Article 15"},
				Category:      "multi-hop",
				Explanation:   "Art. 15(1)-(3).",
			},
			{
				Question:      "How does GDPR regulate the transfer of personal data in the absence of an adequacy decision?",
				ExpectedFacts: []string{"appropriate safeguards|binding corporate rules|standard data protection clauses|approved code of conduct|approved certification mechanism", "enforceable data subject rights|effective legal remedies", "Article 46"},
				Category:      "multi-hop",
				Explanation:   "Art. 46(1)-(2).",
			},
			{
				Question:      "What are the conditions for processing personal data based on legitimate interests?",
				ExpectedFacts: []string{"legitimate interests pursued by the controller or by a third party|necessary|not overridden by the interests or fundamental rights|data subject", "Article 6|balancing test"},
				Category:      "multi-hop",
				Explanation:   "Art. 6(1)(f) + recital 47.",
			},
		},
	}
}

// GDPRHardDataset returns 30 hard (synthesis / regulatory-chain) test cases
// requiring deep understanding of interconnected GDPR provisions.
func GDPRHardDataset() Dataset {
	return Dataset{
		Name:       "GDPR Hard - Synthesis / Regulatory Chain",
		Difficulty: DifficultyHard,
		Tests: []TestCase{
			{
				Question:      "Trace the complete lifecycle of a data breach from detection through resolution under GDPR, referencing all applicable articles.",
				ExpectedFacts: []string{"detection|72 hours|supervisory authority", "high risk|data subject notification", "documentation|Article 33|Article 34|Article 5"},
				Category:      "synthesis",
				Explanation:   "Arts. 33(1) notification, 33(5) documentation, 34 data subject notification, 5(1)(f) integrity.",
			},
			{
				Question:      "How do the principles of purpose limitation and data minimisation interact with the right to data portability?",
				ExpectedFacts: []string{"purpose limitation|data minimisation", "portability|structured, commonly used and machine-readable", "Article 5|Article 20|provided to a controller|consent or contract"},
				Category:      "synthesis",
				Explanation:   "Art. 5(1)(b)-(c) principles vs Art. 20 scope limitation to consent/contract.",
			},
			{
				Question:      "Explain how GDPR's accountability principle connects to DPIAs, DPOs, records of processing, and codes of conduct.",
				ExpectedFacts: []string{"accountability|demonstrate compliance", "DPIA|data protection officer|records of processing activities|codes of conduct", "Article 5|Article 30|Article 35|Article 37|Article 40"},
				Category:      "synthesis",
				Explanation:   "Accountability chain: Art. 5(2) -> Art. 30, 35, 37, 40.",
			},
			{
				Question:      "If a data subject exercises their right to erasure, what obligations flow to processors and third parties?",
				ExpectedFacts: []string{"erasure|inform processors|taken reasonable steps", "third parties|links, copies or replications", "Article 17|Article 19|without undue delay"},
				Category:      "synthesis",
				Explanation:   "Art. 17(2) notification to third parties + Art. 19 notification obligation.",
			},
			{
				Question:      "How does GDPR's regulation of cross-border processing interact with the consistency mechanism and mutual assistance?",
				ExpectedFacts: []string{"cross-border processing|lead supervisory authority|one-stop-shop", "consistency mechanism|mutual assistance", "Article 4|Article 56|Article 60|Article 61|Article 63"},
				Category:      "synthesis",
				Explanation:   "Arts. 56 (lead), 60 (cooperation), 61 (mutual assistance), 63 (consistency).",
			},
			{
				Question:      "Analyze how GDPR balances the protection of personal data with freedom of expression and information.",
				ExpectedFacts: []string{"freedom of expression|journalistic purposes|academic|artistic|literary", "exemptions or derogations|Article 85|reconcile|Member States shall by law"},
				Category:      "synthesis",
				Explanation:   "Art. 85(1)-(2) requires Member States to reconcile data protection with expression.",
			},
			{
				Question:      "How do the conditions for valid consent differ when processing special categories of data versus ordinary data?",
				ExpectedFacts: []string{"consent|explicit consent", "special categories|racial or ethnic origin|political opinions|religious|genetic|biometric|health|sex life", "Article 6|Article 9"},
				Category:      "synthesis",
				Explanation:   "Art. 6(1)(a) consent vs Art. 9(2)(a) explicit consent for special categories.",
			},
			{
				Question:      "Trace the governance hierarchy from a national supervisory authority to the European Data Protection Board to the Court of Justice.",
				ExpectedFacts: []string{"supervisory authority|European Data Protection Board|Court of Justice", "consistency mechanism|binding decision|judicial remedy", "Article 58|Article 65|Article 78"},
				Category:      "synthesis",
				Explanation:   "Arts. 58 (SA powers), 65 (Board binding decisions), 78 (judicial remedy against SA).",
			},
			{
				Question:      "How does GDPR handle conflicts between the right of access and trade secrets or intellectual property?",
				ExpectedFacts: []string{"right of access|not adversely affect the rights and freedoms of others", "trade secrets|intellectual property", "Article 15|Recital 63"},
				Category:      "synthesis",
				Explanation:   "Art. 15(4) + Recital 63: access should not adversely affect others' rights.",
			},
			{
				Question:      "How do the provisions on automated decision-making interact with the transparency and fairness principles?",
				ExpectedFacts: []string{"automated decision-making|profiling", "transparency|meaningful information about the logic involved|significance|envisaged consequences", "Article 5|Article 13|Article 14|Article 22"},
				Category:      "synthesis",
				Explanation:   "Arts. 13(2)(f), 14(2)(g) transparency + Art. 22 restrictions + Art. 5(1)(a) fairness.",
			},
			{
				Question:      "What is the complete chain of safeguards for international data transfers when no adequacy decision exists?",
				ExpectedFacts: []string{"appropriate safeguards|binding corporate rules|standard contractual clauses", "code of conduct|certification|derogations|explicit consent|public interest|legal claims", "Article 46|Article 47|Article 49"},
				Category:      "synthesis",
				Explanation:   "Arts. 46 (safeguards), 47 (BCRs), 49 (derogations).",
			},
			{
				Question:      "How does GDPR's security obligation in Article 32 relate to the breach notification requirements and the accountability principle?",
				ExpectedFacts: []string{"security of processing|appropriate technical and organisational measures|pseudonymisation|encryption", "confidentiality|integrity|availability|resilience", "breach notification|accountability|Article 5|Article 32|Article 33"},
				Category:      "synthesis",
				Explanation:   "Art. 32 security -> Art. 33 breach consequence -> Art. 5(2) accountability.",
			},
			{
				Question:      "How do records of processing activities support the roles of both controllers and processors?",
				ExpectedFacts: []string{"records of processing activities|controller|processor", "name and contact details|purposes|categories of data subjects|categories of processing", "Article 30"},
				Category:      "synthesis",
				Explanation:   "Art. 30(1) controller records vs Art. 30(2) processor records.",
			},
			{
				Question:      "Explain how the right to object interacts with direct marketing and processing based on legitimate interests.",
				ExpectedFacts: []string{"right to object|direct marketing|profiling", "legitimate interests|no longer processed|compelling legitimate grounds|override", "Article 21"},
				Category:      "synthesis",
				Explanation:   "Art. 21(1) general objection + Art. 21(2)-(3) absolute right for direct marketing.",
			},
			{
				Question:      "How does GDPR regulate data processing in the employment context?",
				ExpectedFacts: []string{"employment context|Member States may|more specific rules", "recruitment|performance of the employment contract|collective agreement", "Article 88"},
				Category:      "synthesis",
				Explanation:   "Art. 88(1)-(2) employment processing.",
			},
			{
				Question:      "How do the different types of administrative fines relate to the severity of GDPR infringements?",
				ExpectedFacts: []string{"10 000 000|2 %", "20 000 000|4 %", "effective, proportionate and dissuasive|Article 83"},
				Category:      "synthesis",
				Explanation:   "Art. 83(4) tier 1 (10M/2%) vs Art. 83(5) tier 2 (20M/4%) + Art. 83(1) principles.",
			},
			{
				Question:      "How does the GDPR address the tension between data retention for archiving purposes and the storage limitation principle?",
				ExpectedFacts: []string{"storage limitation|archiving purposes in the public interest|scientific or historical research|statistical", "appropriate safeguards|Article 5|Article 89"},
				Category:      "synthesis",
				Explanation:   "Art. 5(1)(e) storage limitation exception for Art. 89(1) purposes.",
			},
			{
				Question:      "How do the provisions on representative of controllers not established in the Union work?",
				ExpectedFacts: []string{"representative|not established in the Union|written mandate", "Member State|offering goods or services|monitoring behaviour", "Article 27|Article 3"},
				Category:      "synthesis",
				Explanation:   "Art. 27(1)-(2) representative + Art. 3(2) territorial scope.",
			},
			{
				Question:      "How do the transparency obligations differ between data collected directly from the data subject and data obtained from other sources?",
				ExpectedFacts: []string{"collected from the data subject|not obtained from the data subject", "identity|purposes|legitimate interests|source|categories", "Article 13|Article 14"},
				Category:      "synthesis",
				Explanation:   "Art. 13 (direct collection) vs Art. 14 (indirect) — Art. 14 adds source disclosure.",
			},
			{
				Question:      "How does the GDPR create a framework for certification mechanisms and their relationship to adequacy of protection?",
				ExpectedFacts: []string{"certification mechanisms|data protection seals and marks|demonstrate compliance", "appropriate safeguards|Article 42|Article 43|Article 46"},
				Category:      "synthesis",
				Explanation:   "Art. 42 (certification) + Art. 43 (certification bodies) + Art. 46(2)(f).",
			},
			{
				Question:      "Trace how a data subject complaint escalates from a supervisory authority to a court under GDPR.",
				ExpectedFacts: []string{"lodge a complaint|supervisory authority", "effective judicial remedy|against a supervisory authority|against a controller or processor", "Article 77|Article 78|Article 79"},
				Category:      "synthesis",
				Explanation:   "Arts. 77 (complaint), 78 (judicial vs SA), 79 (judicial vs controller/processor).",
			},
			{
				Question:      "How does GDPR regulate processing of biometric data and genetic data as special categories?",
				ExpectedFacts: []string{"biometric data|genetic data|special categories|prohibited", "explicit consent|employment|substantial public interest", "Article 4|Article 9"},
				Category:      "synthesis",
				Explanation:   "Art. 4(13)-(14) definitions + Art. 9(1)-(2) processing conditions.",
			},
			{
				Question:      "How does the right to restriction of processing interact with the obligations to notify recipients?",
				ExpectedFacts: []string{"restriction of processing|marking|storage|data subject's consent", "inform recipients|notify the data subject before lifting", "Article 18|Article 19"},
				Category:      "synthesis",
				Explanation:   "Art. 18 (restriction) + Art. 19 (notification obligation regarding restriction).",
			},
			{
				Question:      "How do the GDPR provisions on delegated acts and implementing acts work?",
				ExpectedFacts: []string{"delegated acts|implementing acts|Commission", "European Parliament|Council", "Article 92|Article 93"},
				Category:      "synthesis",
				Explanation:   "Art. 92 (delegated acts) + Art. 93 (committee procedure).",
			},
			{
				Question:      "How does GDPR address the relationship between national security, defence, and data protection?",
				ExpectedFacts: []string{"national security|defence", "not within the scope|does not apply|outside the scope", "Article 2|Article 23", "restrictions|Member States may restrict"},
				Category:      "synthesis",
				Explanation:   "Art. 2(2)(a) material scope exclusions + Art. 23 restrictions for national security.",
			},
			{
				Question:      "How do the prior consultation requirements with the supervisory authority work when a DPIA indicates high risk?",
				ExpectedFacts: []string{"prior consultation|supervisory authority|high risk|DPIA", "cannot be sufficiently mitigated|written advice|8 weeks", "Article 36"},
				Category:      "synthesis",
				Explanation:   "Art. 36(1)-(2) prior consultation.",
			},
			{
				Question:      "Explain the interplay between legitimate interests, the right to object, and the balancing test in practice.",
				ExpectedFacts: []string{"legitimate interests|balancing test|right to object", "compelling legitimate grounds|override|interests, rights and freedoms", "Article 6|Article 21|Recital 47"},
				Category:      "synthesis",
				Explanation:   "Art. 6(1)(f) + Art. 21(1) + Recital 47 reasonable expectations.",
			},
			{
				Question:      "How does GDPR create a layered enforcement system involving supervisory authorities, the Board, and national courts?",
				ExpectedFacts: []string{"supervisory authority|Board|national courts", "consistency mechanism|binding decision|effective judicial remedy|administrative fines", "Article 58|Article 65|Article 78|Article 83"},
				Category:      "synthesis",
				Explanation:   "Multi-level enforcement: SA (Art. 58), Board (Art. 65), courts (Art. 78), fines (Art. 83).",
			},
			{
				Question:      "How do the derogations for specific situations in Chapter IX interact with the general processing principles?",
				ExpectedFacts: []string{"Chapter IX|freedom of expression|employment|archiving|research|statistical|churches", "reconcile|derogation|Article 85|Article 89"},
				Category:      "synthesis",
				Explanation:   "Chapter IX (Arts. 85-91) provides derogations from general rules.",
			},
			{
				Question:      "How does GDPR address the concept of risk-based approach across all of its provisions?",
				ExpectedFacts: []string{"risk-based|high risk|likelihood and severity", "DPIA|prior consultation|breach notification|security measures", "Article 24|Article 32|Article 33|Article 35|Article 36"},
				Category:      "synthesis",
				Explanation:   "Risk appears in Arts. 24, 32, 33, 34, 35, 36.",
			},
		},
	}
}

// GDPRSuperHardDataset returns 50 super-hard (adversarial / polysemy / edge-case)
// test cases designed to probe subtle distinctions and edge cases in the GDPR.
func GDPRSuperHardDataset() Dataset {
	return Dataset{
		Name:       "GDPR Super-Hard - Adversarial / Edge Cases",
		Difficulty: DifficultySuperHard,
		Tests: []TestCase{
			{
				Question:      "Can a data subject exercise the right to data portability for data processed based on legitimate interests?",
				ExpectedFacts: []string{"portability|only applies|consent|contract|not available for legitimate interests", "Article 20"},
				Category:      "adversarial",
				Explanation:   "Art. 20(1) limits portability to consent or contract basis only.",
			},
			{
				Question:      "If a controller processes data for both marketing and fraud prevention, can a data subject's objection to marketing affect the fraud prevention processing?",
				ExpectedFacts: []string{"right to object|direct marketing|absolute right", "fraud prevention|legitimate interests|separate processing purposes", "Article 21"},
				Category:      "adversarial",
				Explanation:   "Art. 21(2)-(3): objection to marketing is absolute but doesn't extend to other lawful bases.",
			},
			{
				Question:      "Does GDPR apply to a deceased person's personal data?",
				ExpectedFacts: []string{"natural person|living individuals|not apply to personal data of deceased persons|Member States may provide", "Recital 27"},
				Category:      "adversarial",
				Explanation:   "Recital 27: GDPR does not apply to deceased persons' data.",
			},
			{
				Question:      "Can a controller refuse the right to erasure if the data is needed for exercising freedom of expression?",
				ExpectedFacts: []string{"right to erasure|does not apply|freedom of expression and information", "Article 17|Article 17(3)(a)"},
				Category:      "adversarial",
				Explanation:   "Art. 17(3)(a) exemption for freedom of expression.",
			},
			{
				Question:      "What happens when a processor detects a personal data breach — must they notify the data subject directly?",
				ExpectedFacts: []string{"processor|notify the controller|without undue delay", "controller notifies|supervisory authority|data subject", "Article 33|Article 34"},
				Category:      "adversarial",
				Explanation:   "Art. 33(2): processor notifies controller, not the data subject directly.",
			},
			{
				Question:      "Under what circumstances can personal data be processed without any legal basis under Article 6?",
				ExpectedFacts: []string{"no circumstances|none|not permitted|not possible|unlawful|cannot be processed|always requires|must have", "legal basis|lawful basis|Article 6"},
				Category:      "adversarial",
				Explanation:   "Art. 6(1) requires at least one legal basis — processing without a basis is unlawful.",
			},
			{
				Question:      "Is consent valid if the data subject is forced to consent as a condition for receiving a service?",
				ExpectedFacts: []string{"freely given|not valid|not conditional|genuine choice|significant imbalance", "Article 7|Recital 42|Recital 43"},
				Category:      "adversarial",
				Explanation:   "Art. 7(4) + Recitals 42-43: consent not free if conditional on service.",
			},
			{
				Question:      "Can a supervisory authority impose a fine on a public authority or body?",
				ExpectedFacts: []string{"public authority|Member States may provide|whether and to what extent administrative fines may be imposed", "Article 83(7)"},
				Category:      "adversarial",
				Explanation:   "Art. 83(7) gives Member States discretion on fining public authorities.",
			},
			{
				Question:      "If an adequacy decision is later invalidated, what happens to data already transferred?",
				ExpectedFacts: []string{"adequacy decision|invalidated|appropriate safeguards", "Article 46|binding corporate rules|standard contractual clauses", "existing transfers|Commission shall repeal, amend or adopt|Article 45(9)"},
				Category:      "adversarial",
				Explanation:   "Art. 45(3)-(9): Commission can revoke; controllers must find alternative basis.",
			},
			{
				Question:      "Can a data subject object to processing carried out by a public authority in the performance of its tasks?",
				ExpectedFacts: []string{"public authority|public interest|right to object", "controller must demonstrate compelling legitimate grounds|override", "Article 6(1)(e)|Article 21(1)"},
				Category:      "adversarial",
				Explanation:   "Art. 21(1) allows objection even to public interest processing, with balancing.",
			},
			{
				Question:      "What is the relationship between 'not for profit body' processing of special category data and the requirement for explicit consent?",
				ExpectedFacts: []string{"not-for-profit body|political, philosophical, religious or trade union aim", "members or former members|no disclosure without consent", "Article 9(2)(d)"},
				Category:      "adversarial",
				Explanation:   "Art. 9(2)(d): not-for-profit can process without explicit consent for members.",
			},
			{
				Question:      "Does GDPR require encryption of personal data?",
				ExpectedFacts: []string{"encryption|not mandatory|appropriate measures|state of the art", "pseudonymisation and encryption|Article 32"},
				Category:      "adversarial",
				Explanation:   "Art. 32(1)(a) lists encryption as one option, not a requirement.",
			},
			{
				Question:      "Can a 13-year-old in Spain consent to an information society service under GDPR?",
				ExpectedFacts: []string{"13 years|Member States may provide|lower age|not below 13 years", "Article 8"},
				Category:      "adversarial",
				Explanation:   "Art. 8(1) allows Member States to lower to 13; Spain could implement this.",
			},
			{
				Question:      "What happens when the right to data portability conflicts with the rights and freedoms of others?",
				ExpectedFacts: []string{"portability|shall not adversely affect the rights and freedoms of others", "Article 20(4)"},
				Category:      "adversarial",
				Explanation:   "Art. 20(4): portability must not adversely affect others' rights.",
			},
			{
				Question:      "Can a processor appoint a sub-processor without the controller's authorisation?",
				ExpectedFacts: []string{"sub-processor|prior specific written authorisation|prior general written authorisation", "Article 28(2)"},
				Category:      "adversarial",
				Explanation:   "Art. 28(2): either specific or general written authorisation required.",
			},
			{
				Question:      "Does the GDPR apply to data processing by EU institutions and bodies?",
				ExpectedFacts: []string{"EU institutions|Regulation (EC) No 45/2001|adapted", "Article 2|Article 98|Recital 17"},
				Category:      "adversarial",
				Explanation:   "Recital 17 + Art. 98: GDPR provides framework; separate regulation for EU institutions.",
			},
			{
				Question:      "Can a controller charge a fee for responding to data subject requests?",
				ExpectedFacts: []string{"free of charge|reasonable fee|manifestly unfounded or excessive|repetitive", "Article 12(5)"},
				Category:      "adversarial",
				Explanation:   "Art. 12(5): free, unless manifestly unfounded/excessive.",
			},
			{
				Question:      "How does GDPR address a situation where processing is necessary for the vital interests of a person who is unconscious?",
				ExpectedFacts: []string{"vital interests|cannot give consent|another natural person|physical integrity", "Article 6(1)(d)|Recital 46"},
				Category:      "adversarial",
				Explanation:   "Art. 6(1)(d) + Recital 46: vital interests when subject cannot consent.",
			},
			{
				Question:      "Can a controller rely on consent obtained before GDPR's application date?",
				ExpectedFacts: []string{"consent|obtained before|conditions|compliant with GDPR|Recital 171", "Article 6|continue"},
				Category:      "adversarial",
				Explanation:   "Recital 171: pre-existing consent valid if GDPR-compliant.",
			},
			{
				Question:      "What is the difference between 'restriction of processing' and 'erasure'?",
				ExpectedFacts: []string{"restriction|marking|storage|limits further processing", "erasure|deletion|destruction", "Article 4(3)|Article 17|Article 18"},
				Category:      "adversarial",
				Explanation:   "Art. 4(3): restriction = marking + limiting; Art. 17: erasure = deletion.",
			},
			{
				Question:      "Can a data subject withdraw consent and simultaneously exercise the right to erasure?",
				ExpectedFacts: []string{"withdraw consent|right to erasure|Article 7|Article 17", "not retroactive|does not affect lawfulness of processing based on consent before withdrawal"},
				Category:      "adversarial",
				Explanation:   "Art. 7(3) withdrawal not retroactive + Art. 17(1)(b) erasure when consent withdrawn.",
			},
			{
				Question:      "Does the GDPR require prior approval from the supervisory authority before processing?",
				ExpectedFacts: []string{"prior approval|not required|prior consultation|DPIA|high risk", "Article 36|accountability"},
				Category:      "adversarial",
				Explanation:   "Art. 36: only prior consultation (not approval) when DPIA shows high risk.",
			},
			{
				Question:      "What are the differences between the two tiers of administrative fines and which violations fall under each?",
				ExpectedFacts: []string{"10 000 000|2 %|controller and processor obligations|certification body|monitoring body", "20 000 000|4 %|basic principles|data subject rights|international transfers", "Article 83(4)|Article 83(5)"},
				Category:      "adversarial",
				Explanation:   "Art. 83(4) tier 1 vs Art. 83(5) tier 2 with specific violation categories.",
			},
			{
				Question:      "Can profiling be used for credit scoring under GDPR, and what safeguards apply?",
				ExpectedFacts: []string{"profiling|automated decision-making|credit scoring", "legal effects|similarly significantly affects|right to explanation|human intervention", "Article 22|Recital 71"},
				Category:      "adversarial",
				Explanation:   "Art. 22 + Recital 71: credit scoring as example of significant effect.",
			},
			{
				Question:      "If two companies are joint controllers but disagree about a data subject request, who is liable?",
				ExpectedFacts: []string{"joint controllers|arrangement|regardless of terms", "data subject may exercise rights against each|Article 26(3)"},
				Category:      "adversarial",
				Explanation:   "Art. 26(3): data subject can exercise against each controller regardless of arrangement.",
			},
			{
				Question:      "What is the minimum number of Board members required for a binding decision?",
				ExpectedFacts: []string{"two-thirds majority|simple majority|rules of procedure", "Article 72"},
				Category:      "adversarial",
				Explanation:   "Art. 72(1)-(2) voting rules for the Board.",
			},
			{
				Question:      "How does GDPR interact with the ePrivacy Directive regarding cookies and tracking?",
				ExpectedFacts: []string{"ePrivacy|Directive 2002/58/EC|electronic communications", "not impose additional obligations|Recital 173|Article 95"},
				Category:      "adversarial",
				Explanation:   "Art. 95 + Recital 173: GDPR does not impose additional obligations on matters covered by ePrivacy.",
			},
			{
				Question:      "Can a data subject request their data be transferred directly from one controller to another?",
				ExpectedFacts: []string{"data portability|right to have the personal data transmitted directly|technically feasible", "Article 20(2)"},
				Category:      "adversarial",
				Explanation:   "Art. 20(2): direct transfer where technically feasible.",
			},
			{
				Question:      "What happens if a controller fails to respond to a data subject request within the time limit?",
				ExpectedFacts: []string{"one month|extended by two further months", "inform the data subject|reasons for the delay|right to lodge a complaint|judicial remedy", "Article 12(3)|Article 12(4)"},
				Category:      "adversarial",
				Explanation:   "Art. 12(3)-(4): 1 month deadline, extendable by 2 months with notification.",
			},
			{
				Question:      "Does the GDPR apply to purely personal or household data processing?",
				ExpectedFacts: []string{"not apply|purely personal or household activity", "Article 2(2)(c)|no connection with a professional or commercial activity|Recital 18"},
				Category:      "adversarial",
				Explanation:   "Art. 2(2)(c) + Recital 18: household exemption.",
			},
			{
				Question:      "How does GDPR address the processing of personal data of children for preventive or counselling services?",
				ExpectedFacts: []string{"child|preventive or counselling services|directly to a child", "consent of the holder of parental responsibility|Article 8|Recital 38"},
				Category:      "adversarial",
				Explanation:   "Recital 38: children merit specific protection; preventive services exception.",
			},
			{
				Question:      "Can a Member State create additional conditions for processing genetic, biometric, or health data?",
				ExpectedFacts: []string{"genetic data|biometric data|health data", "Member States may maintain or introduce further conditions|limitations", "Article 9(4)"},
				Category:      "adversarial",
				Explanation:   "Art. 9(4): Member States can add conditions for genetic/biometric/health data.",
			},
			{
				Question:      "What is the difference between a 'supervisory authority concerned' and the 'lead supervisory authority'?",
				ExpectedFacts: []string{"lead supervisory authority|main establishment", "supervisory authority concerned|substantially affected|establishment in that Member State|data subjects in that Member State", "Article 4(22)|Article 4(23)|Article 56"},
				Category:      "adversarial",
				Explanation:   "Art. 4(22) lead vs Art. 4(23) concerned authority definitions.",
			},
			{
				Question:      "Can a controller process personal data for a new purpose without obtaining new consent?",
				ExpectedFacts: []string{"further processing|compatible purpose|compatibility", "not require a new legal basis|same legal basis|without new consent", "Article 6(4)|Article 6"},
				Category:      "adversarial",
				Explanation:   "Art. 6(4) compatibility test for new purposes.",
			},
			{
				Question:      "How does GDPR apply to research involving historical archives?",
				ExpectedFacts: []string{"archiving purposes in the public interest|scientific or historical research", "derogations|storage limitation|data minimisation", "Article 5|Article 89|Recital 156"},
				Category:      "adversarial",
				Explanation:   "Art. 89 + Recital 156: safeguards for archiving/research.",
			},
			{
				Question:      "Can a data subject bring a claim for compensation against a processor directly?",
				ExpectedFacts: []string{"processor|compensation|liable for damage", "shall be exempt only if it proves not responsible", "Article 82"},
				Category:      "adversarial",
				Explanation:   "Art. 82(2)-(3): processor directly liable for damage from GDPR breach.",
			},
			{
				Question:      "What happens to BCR approvals when the GDPR came into force and replaced the previous directive?",
				ExpectedFacts: []string{"binding corporate rules|approved under Directive 95/46/EC", "amended where necessary|continue to be valid", "Article 46(5)|Recital 171"},
				Category:      "adversarial",
				Explanation:   "Art. 46(5) + Recital 171: pre-GDPR BCRs continue if amended as needed.",
			},
			{
				Question:      "How does GDPR handle the scenario where a data subject requests access but the controller cannot identify the data subject?",
				ExpectedFacts: []string{"unable to identify|not apply Articles 15 to 20", "additional information|demonstrate identity", "Article 11|Article 12"},
				Category:      "adversarial",
				Explanation:   "Art. 11(2) + Art. 12(6): controller not obliged if cannot identify subject.",
			},
			{
				Question:      "Under what circumstances can processing of special category data occur without the data subject's knowledge?",
				ExpectedFacts: []string{"substantial public interest|proportionate|essence of the right to data protection|suitable and specific measures", "Article 9(2)(g)|public health|Article 9(2)(i)"},
				Category:      "adversarial",
				Explanation:   "Art. 9(2)(g) substantial public interest + Art. 9(2)(i) public health.",
			},
			{
				Question:      "How does the GDPR address representative actions on behalf of data subjects?",
				ExpectedFacts: []string{"not-for-profit body|mandate of the data subject", "lodge complaint|exercise rights|receive compensation", "Article 80"},
				Category:      "adversarial",
				Explanation:   "Art. 80(1)-(2): representation by not-for-profit bodies.",
			},
			{
				Question:      "Can a supervisory authority suspend data flows to a third country even when an adequacy decision exists?",
				ExpectedFacts: []string{"suspend|urgency procedure|provisional measures|territory|three months", "Article 66|adequacy decision|Court of Justice"},
				Category:      "adversarial",
				Explanation:   "Art. 66: urgent procedure for provisional measures; only CJEU can invalidate adequacy.",
			},
			{
				Question:      "What is the interplay between the right to rectification and the obligation to notify third parties?",
				ExpectedFacts: []string{"rectification|notify each recipient|disclosed to", "unless impossible or disproportionate effort", "Article 16|Article 19"},
				Category:      "adversarial",
				Explanation:   "Art. 19: controller must notify recipients of rectification.",
			},
			{
				Question:      "Does GDPR apply to a company outside the EU that only monitors EU residents' behavior online?",
				ExpectedFacts: []string{"monitoring of their behaviour|behaviour takes place within the Union", "Article 3(2)(b)|territorial scope|not established in the Union"},
				Category:      "adversarial",
				Explanation:   "Art. 3(2)(b): GDPR applies to monitoring of EU data subjects' behavior.",
			},
			{
				Question:      "How does GDPR address the conflict between the right to be forgotten and the public's right to information?",
				ExpectedFacts: []string{"right to erasure|freedom of expression and information|public interest", "archiving|research|exercise or defence of legal claims", "Article 17(3)"},
				Category:      "adversarial",
				Explanation:   "Art. 17(3)(a)-(e) lists exemptions including expression and public interest.",
			},
			{
				Question:      "What is the 'household exemption' and does it apply to personal data shared on social media?",
				ExpectedFacts: []string{"purely personal or household activity|Article 2(2)(c)", "social networking|Recital 18|extends beyond a purely personal or household activity"},
				Category:      "adversarial",
				Explanation:   "Recital 18: social networking may extend beyond household exemption.",
			},
			{
				Question:      "Can a data subject exercise the right to restriction of processing during a dispute about data accuracy?",
				ExpectedFacts: []string{"restriction|accuracy contested|period enabling the controller to verify accuracy", "Article 18(1)(a)"},
				Category:      "adversarial",
				Explanation:   "Art. 18(1)(a): restriction during accuracy verification.",
			},
			{
				Question:      "How does GDPR address the processing of personal data in the context of religious organisations?",
				ExpectedFacts: []string{"churches|religious associations|comprehensive rules", "protection of natural persons|applied in accordance with GDPR", "Article 91"},
				Category:      "adversarial",
				Explanation:   "Art. 91(1)-(2): existing religious body rules can continue if GDPR-consistent.",
			},
			{
				Question:      "What role does the European Data Protection Supervisor play under GDPR?",
				ExpectedFacts: []string{"European Data Protection Supervisor|EDPS", "participate|member of the Board|activities of the Board", "Article 68|Recital 139"},
				Category:      "adversarial",
				Explanation:   "Art. 68(3): EDPS has right to participate in Board activities.",
			},
			{
				Question:      "Can a controller lawfully process personal data of non-EU citizens who are physically present in the EU?",
				ExpectedFacts: []string{"territorial scope|data subjects who are in the Union|regardless of citizenship", "offering goods or services|monitoring behaviour", "Article 3"},
				Category:      "adversarial",
				Explanation:   "Art. 3(2): applies to data subjects 'in the Union' regardless of nationality.",
			},
			{
				Question:      "How does GDPR address the use of personal data for electoral activities and political campaigns?",
				ExpectedFacts: []string{"electoral activities|political opinions|special category", "public interest|democratic process|Recital 56", "Article 9"},
				Category:      "adversarial",
				Explanation:   "Recital 56: processing for electoral activities may be in public interest.",
			},
		},
	}
}

// GDPRAllDatasets returns all GDPR evaluation datasets keyed by difficulty level.
func GDPRAllDatasets() map[string]Dataset {
	return map[string]Dataset{
		DifficultyEasy:      GDPREasyDataset(),
		DifficultyMedium:    GDPRMediumDataset(),
		DifficultyHard:      GDPRHardDataset(),
		DifficultySuperHard: GDPRSuperHardDataset(),
	}
}
