export async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(path)
  if (!response.ok) {
    let message = response.statusText
    try {
      const body = await response.json() as { error?: string }
      if (body.error) message = body.error
    } catch {
      // keep status text
    }
    throw new Error(message)
  }
  return await response.json() as T
}

export async function getText(path: string): Promise<string> {
  const response = await fetch(path)
  if (!response.ok) {
    throw new Error(response.statusText)
  }
  return await response.text()
}
