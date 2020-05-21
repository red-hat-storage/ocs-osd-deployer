# FROM centos:8
FROM registry.access.redhat.com/ubi8/ubi-minimal

COPY k8s.repo /etc/yum.repos.d/k8s.repo
RUN microdnf update -y && \
    microdnf install -y \
      bash \
      kubectl \
    && microdnf clean all && \
    rm -rf /var/cache/yum

# COPY volrecycler/kubectl-sa.sh /
# COPY volrecycler/recycler.sh /
# COPY volrecycler/locker.sh /

# RUN chmod 755 /kubectl-sa.sh \
#               /locker.sh \
#               /recycler.sh

ARG builddate="(unknown)"
ARG version="(unknown)"

LABEL org.label-schema.build-date="${builddate}" \
      org.label-schema.description="Pod to deploy the initial StorageCluster yaml for OCS" \
      org.label-schema.license="Apache-2.0" \
      org.label-schema.name="OCS on OSD deployer" \
      org.label-schema.schema-version="1.0" \
      org.label-schema.vcs-ref="${version}" \
      org.label-schema.vcs-url="https://github.com/johnstrunk/ocs-osd-deployer" \
      org.label-schema.vendor="Red Hat" \
      org.label-schema.version="${version}"

ENTRYPOINT [ "/locker.sh" ]
