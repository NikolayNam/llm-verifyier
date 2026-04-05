import os
from google import genai
from mistralai.client import Mistral


class Runner:
    def __init__(self) -> None:
        self.google = genai.Client()
        self.mistral = Mistral(api_key=os.environ["MISTRAL_API_KEY"])

    def generate(self, provider: str, model: str, prompt: str) -> str:
        if provider == "google":
            resp = self.google.models.generate_content(
                model=model,
                contents=prompt,
            )
            return resp.text

        if provider == "mistral":
            resp = self.mistral.chat.complete(
                model=model,
                messages=[{"role": "user", "content": prompt}],
            )
            return resp.choices[0].message.content

        raise ValueError(f"Unknown provider: {provider}")
