export interface HeroSection {
  title: string
  subtitle: string
  ctaPrimary: string
  ctaSecondary: string
}

export interface Service {
  id: string
  title: string
  description: string
  icon: string
}

export interface Solution {
  id: string
  name: string
  status: string
  summary: string
}

export interface ChatService {
  title: string
  description: string
  features: string[]
  techStack: string[]
}

export interface TrustMetric {
  label: string
  value: string
}

export interface TrustSection {
  metrics: TrustMetric[]
  badges: string[]
}

export interface ProcessStep {
  step: number
  title: string
  description: string
}

export interface AboutSection {
  mission: string
  vision: string
  description: string
}

export interface ContactSection {
  email: string
  address: string
  phone: string
}

export interface Content {
  hero: HeroSection
  services: Service[]
  solutions: Solution[]
  chatService: ChatService
  trust: TrustSection
  process: ProcessStep[]
  about: AboutSection
  contact: ContactSection
}

export interface Inquiry {
  id: string
  name: string
  email: string
  message: string
  createdAt: string
}

export interface APIResponse<T = unknown> {
  code: number
  message: string
  data?: T
}
