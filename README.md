# How to use it

Step 1 (Optional): Ideally run this in a Python virtual environment. This documentation won't go into the details of setting up a Python virtual environment.

Step 2: Install the requirements

```
pip install -r requirements.txt
```

Step 3: Convert the notebook to slides

```
jupyter nbconvert effective-o11y-on-kubernetes.ipynb --to slides --post serve
```

Step 4: Run demo

```bash
(
    cd demo
    make docker-build docker-push deploy # into your k8s cluster
)
```
