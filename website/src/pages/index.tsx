import React from 'react';
import clsx from 'clsx';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import useBaseUrl from '@docusaurus/useBaseUrl';
import styles from './index.module.css';
import HomepageFeatures from '../components/HomepageFeatures';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  const logoUrl = useBaseUrl('/img/beluga-logo.svg');
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <div className={styles.heroContent}>
          <div className={styles.heroImage}>
            <img 
              src={logoUrl} 
              alt="Beluga AI Framework Logo" 
              className={styles.logo}
            />
          </div>
          <h1 className="hero__title">{siteConfig.title}</h1>
          <p className="hero__subtitle">
            A production-ready Go framework for building sophisticated AI and agentic applications.
            <br />
            <span className={styles.taglineHighlight}>
              Enterprise-grade • Extensible • Observable
            </span>
          </p>
          <div className={styles.buttons}>
            <Link
              className="button button--secondary button--lg"
              to="/docs/quickstart">
              Get Started →
            </Link>
            <Link
              className="button button--outline button--secondary button--lg"
              to="/docs/README">
              Read Documentation
            </Link>
          </div>
        </div>
      </div>
    </header>
  );
}

function HomepageDescription() {
  return (
    <section className={styles.description}>
      <div className="container">
        <div className="row">
          <div className="col col--8 col--offset-2">
            <div className={styles.descriptionContent}>
              <h2>What is Beluga AI Framework?</h2>
              <p>
                <strong>Beluga AI Framework</strong> is a comprehensive, production-ready framework written in Go, 
                designed for building sophisticated AI and agentic applications. Inspired by frameworks like 
                LangChain and CrewAI, Beluga AI provides a robust set of tools and abstractions to streamline 
                the development of applications leveraging Large Language Models (LLMs).
              </p>
              <p>
                The framework offers a Go-native, performant, and flexible alternative for creating complex AI workflows. 
                Built with extensibility at its core, Beluga AI empowers Go developers to build next-generation AI 
                applications with enterprise-grade observability, comprehensive testing, and production-ready patterns.
              </p>
              <div className={styles.highlightBox}>
                <p>
                  <strong>🚀 Production Ready:</strong> Beluga AI has completed comprehensive standardization and is 
                  now enterprise-grade with consistent patterns, extensive testing, and production-ready observability 
                  across all 14 packages. The framework is ready for large-scale deployment and development teams.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

export default function Home(): JSX.Element {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={`${siteConfig.title} - Go Framework for AI Applications`}
      description="A production-ready Go framework for building sophisticated AI and agentic applications. Enterprise-grade, extensible, and observable.">
      <HomepageHeader />
      <main>
        <HomepageDescription />
        <HomepageFeatures />
      </main>
    </Layout>
  );
}
