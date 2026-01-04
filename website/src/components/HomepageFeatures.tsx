/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */
import React from 'react';
import clsx from 'clsx';
import styles from './HomepageFeatures.module.css';

type FeatureItem = {
  title: string;
  image: string;
  description: JSX.Element;
};

const FeatureList: FeatureItem[] = [
  {
    title: '🚀 Extensible LLM Integration',
    image: '/img/undraw_docusaurus_mountain.svg',
    description: (
      <>
        Seamlessly connect to various LLM providers (OpenAI, Anthropic, Google Gemini, 
        AWS Bedrock, Ollama, Cohere) with a unified interface. Switch providers without changing 
        your code using our global registry pattern.
      </>
    ),
  },
  {
    title: '🤖 Agent Framework',
    image: '/img/undraw_docusaurus_tree.svg',
    description: (
      <>
        Build autonomous agents capable of reasoning, planning, and executing tasks. 
        Includes ReAct agents, tool integration, and memory management for sophisticated 
        AI applications with comprehensive testing infrastructure.
      </>
    ),
  },
  {
    title: '📊 Production Ready',
    image: '/img/undraw_docusaurus_react.svg',
    description: (
      <>
        Enterprise-grade observability with OpenTelemetry, comprehensive testing, 
        structured logging, metrics, and distributed tracing. Built for large-scale 
        deployment with 100% package standardization.
      </>
    ),
  },
  {
    title: '🔍 RAG Pipeline',
    image: '/img/undraw_docusaurus_mountain.svg',
    description: (
      <>
        Implement Retrieval-Augmented Generation with swappable components for data loading, 
        splitting, embedding, and retrieval. Support for multiple vector stores including 
        pgvector, Pinecone, and Weaviate with global factory patterns.
      </>
    ),
  },
  {
    title: '⚙️ Flexible Orchestration',
    image: '/img/undraw_docusaurus_tree.svg',
    description: (
      <>
        Define and manage complex workflows with a flexible engine. Event-driven architecture 
        with worker pools, retry mechanisms, and circuit breakers for reliable execution 
        with OTEL metrics.
      </>
    ),
  },
  {
    title: '🎤 Voice Agents',
    image: '/img/undraw_docusaurus_react.svg',
    description: (
      <>
        Build natural voice interactions with speech-to-text, text-to-speech, voice activity 
        detection, turn detection, and complete session management. Support for multiple 
        voice providers with streaming capabilities.
      </>
    ),
  },
];

function Feature({title, image, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        <img className={styles.featureSvg} alt={title} src={image} />
      </div>
      <div className="text--center padding-horiz--md">
        <h3>{title}</h3>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): JSX.Element {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          <div className="col col--12">
            <div className="text--center margin-bottom--lg">
              <h2>Key Features</h2>
              <p className={styles.featuresSubtitle}>
                Everything you need to build production-ready AI applications in Go
              </p>
            </div>
          </div>
        </div>
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
