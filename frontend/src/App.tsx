import React from 'react';
import './App.css';
import 'tachyons/css/tachyons.min.css';
import { BrowserRouter as Router, Switch, Route, Link } from 'react-router-dom';
import GAListener from './tracker';
import Search from './Search';
import About from './About';
import Fit from './Fit';
import Fits from './Fits';
import Saved from './Saved';

export default function App() {
	const trackingId =
		process.env.NODE_ENV === 'production' ? 'UA-154150569-1' : undefined;

	return (
		<Router>
			<GAListener trackingId={trackingId}>
				<div className="sans-serif">
					<nav className="pa3 bg-dp04">
						<ul className="list ma0 pa0">
							<li className="ma2">
								<Link to="/">fittin.gs</Link>
							</li>
							<li className="ma2">
								<Link to="/search">search</Link>
							</li>
							<li className="ma2">
								<Link to="/about">about</Link>
							</li>
						</ul>
					</nav>
					<div className="ma1">
						<Switch>
							<Route path="/fit/:id" children={<Fit />} />
							<Route path="/search" children={<Search />} />
							<Route path="/about" children={<About />} />
							<Route path="/saved" children={<Saved />} />
							<Route path="/" children={<Fits />} />
						</Switch>
					</div>
				</div>
			</GAListener>
		</Router>
	);
}
