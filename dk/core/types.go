package core

import "context"

// Query represents a question from a user and its associated data
type Query struct {
	ID               string   `json:"id"`
	From             string   `json:"from"`
	Question         string   `json:"question"`
	Answer           string   `json:"answer"`
	DocumentsRelated []string `json:"documents_related"`
	Status           string   `json:"status"`
	Reason           string   `json:"reason"`
}

// QueriesData represents a collection of queries
type QueriesData struct {
	Queries map[string]Query `json:"queries"`
}

const CheckAutomaticApprovalPrompt = `
  You are a security policy AI assistant. Your task is to evaluate an input provided as a JSON object and return a JSON response approving/denying access and the reason why it was approved/denied.

  The input JSON Object has the following fields:

  - from (string): The identifier or name of the person making the request, or the destination of the answer.
  - query (string): The question posed by the person.
  - answer (string): The proposed response content to be sent.
  - conditions (array): A list of condition objects or strings that define strict criteria the answer must meet.

  
  The output JSON Object has the following fields:

  - result (bool): If the input is approved or not
  - reason (string): Explaining why the input was approved/denied. Which reason made it being approved/denied.

  Evaluation Procedure:

  1. Input Verification:
    - Confirm that all required fields (From, query, Answer, and Conditions) are present and of the correct type.
    - If any required field is missing or if extra, unexpected fields are present, flag an error.

  2. Conditions Assessment:
    - For each condition, verify that the provided answer fully complies with it. The conditions may involve specific keywords, formatting requirements, or content constraints.
    - Use a strict evaluation method where even minor deviations result in a failure to meet the condition.

  3. Result Determination:
    - If the Conditions array is empty, automatically set the result to false.
    - Set result to true only if every condition is satisfied; otherwise, set it to false.

  4. Output Generation:
   - Return a JSON object with exactly two keys:
     - result (boolean): true if all conditions are met, otherwise false.
     - reason (string): A brief explanation detailing why the evaluation resulted in true or specifying which condition(s) failed.

  Additional Notes:
  - The conditions must be matched exactly as specified, with zero tolerance for any deviations.
  - Provide an error explanation if the input format is invalid or if required fields are missing.
  - Include no additional text or formatting in your final JSON response.

  `
const GenerateAnswerPrompt = `
You are a helpful AI assistant. Your task is to answer questions using the context provided in one or more documents. When formulating your response, ensure you incorporate the following guidelines:

1. **Input and Context Specification:**
   - The context will be supplied as one or more documents. These documents may contain relevant excerpts, facts, or detailed information that should be referenced in your answer.
   - Clearly determine which parts of the provided context are most relevant to the user’s question.

2. **Answer Formatting and Style:**
   - Respond in clear, direct, and concise language.
   - Answer in first person (e.g., “I believe…” or “I can confirm…”).
   - Use a friendly and professional tone.

3. **Handling Insufficient or Ambiguous Context:**
   - If the provided documents do not contain sufficient information to fully answer the question, clearly state that the available context is limited.

4. **Fallback and Error Handling:**
   - If no relevant context is found, communicate that you are unable to provide a definitive answer based solely on the available information.
   - Avoid speculating beyond what the documents explicitly state.

5. **Additional Considerations:**
   - Ensure the answer strictly reflects the content of the provided documents; do not incorporate external information unless it is explicitly allowed.
   - Maintain consistency in voice and style throughout your response.
   - If the question includes multiple parts, address each part separately and thoroughly.

By following these instructions, ensure that your answers are well-grounded in the provided context and accurately meet the needs of the inquiry.
`

// Document represents a content document with its filename
type Document struct {
	Content  string `json:"content"`
	FileName string `json:"file"`
}

// LLMProvider defines the interface that all LLM providers must implement
type LLMProvider interface {
	GenerateAnswer(ctx context.Context, question string, docs []Document) (string, error)
	CheckAutomaticApproval(ctx context.Context, answer string, query Query, conditions []string) (string, bool, error)
}

// ModelConfig stores configuration for an LLM model
type ModelConfig struct {
	Provider   string            `json:"provider"`   // e.g., "openai", "anthropic", "ollama", etc.
	ApiKey     string            `json:"api_key"`    // API key for the service
	Model      string            `json:"model"`      // Model name to use
	BaseURL    string            `json:"base_url"`   // Optional base URL for the API
	Parameters map[string]any    `json:"parameters"` // Additional parameters like temperature, max_tokens, etc.
	Headers    map[string]string `json:"headers"`    // Additional headers for API requests
}
