import { sleep, check } from 'k6';
import http from 'k6/http';

export const options = {
    scenarios: {
        feed: {
            executor: 'ramping-vus',
            startVUs: 0,
            gracefulStop: '30s',
            stages: [
                { target: 1000, duration: '20s' },
                { target: 2000, duration: '20s' },
                { target: 1000, duration: '20s' },
            ],
            gracefulRampDown: '30s',
            exec: 'feed',
        },
    },
}

export function feed() {
    let res = http.get('http://127.0.0.1:37000/douyin/feed?');

    let jsonResponse = JSON.parse(res.body);
    check(jsonResponse, {
        'status_code is 0': (json) => json.status_code === 0,
    });

    sleep(3);
}
