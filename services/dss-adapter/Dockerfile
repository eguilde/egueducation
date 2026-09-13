FROM eclipse-temurin:21-jre-jammy
RUN useradd --system --uid 10001 --create-home dss
WORKDIR /opt/dss
COPY target/dss-adapter-0.1.0-SNAPSHOT.jar /opt/dss/dss-adapter.jar
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["java", "-XX:MaxRAMPercentage=75", "-Djava.io.tmpdir=/tmp", "-jar", "/opt/dss/dss-adapter.jar"]
